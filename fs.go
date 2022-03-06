package ydfs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// ydfs implements FS interface
type ydfs struct {
	client *apiclient // api client
	path   string     // base path
	issub  bool       // is this a sub FS?
}

// New returns ydfs.FS which is compliant with
// standard library's fs.FS interface. Token is required for authorization.
// Pre-configured http.Client can be supplied (e.g. with timeout set to specific value).
// If client is nil then http.DefaultClient is used.
func New(token string, client *http.Client) (FS, error) {
	if client == nil {
		client = http.DefaultClient
	}
	c := newApiClient(token, client)
	// checking whether we can fetch disk metadata to
	// make sure that token is valid and we we can send
	// requests to the API.
	if _, err := c.getDiskInfo(); err != nil {
		return nil, err
	}
	return &ydfs{client: c, path: "/", issub: false}, nil
}

func (y *ydfs) newfile(path string) *ydfile {
	return &ydfile{client: y.client, path: path}
}

func (y *ydfs) open(name string) (*ydfile, error) {
	file := y.newfile(y.getFullPath(name))
	if err := file.update(); err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return file, nil
}

func (y *ydfs) getFullPath(name string) string {
	if y.issub {
		name = path.Join(y.path, name)
	}
	return name
}

func (y *ydfs) trimRootPath(name string) string {
	if y.issub {
		name = name[len(y.path):]
	}
	return name
}

func (y *ydfs) Create(name string) (fs.File, error) {
	return nil, nil
}

// Open implements fs.Fs interface
func (y *ydfs) Open(name string) (fs.File, error) {
	return y.open(name)
}

// Stat implements fs.StatFS
func (y *ydfs) Stat(name string) (fs.FileInfo, error) {
	return y.open(name)
}

// Sub implements fs.SubFS
func (y *ydfs) Sub(dir string) (FS, error) {
	fullname := y.getFullPath(dir)
	res, err := y.client.getResourceMinTraffic(fullname)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: dir, Err: err}
	}
	if res.Type != "dir" {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fmt.Errorf("not a directory")}
	}
	normalizeResource(&res)
	return &ydfs{client: y.client, path: res.Path, issub: true}, nil
}

// ReadFile implements fs.ReadFileFS
func (y *ydfs) ReadFile(name string) ([]byte, error) {
	fullname := y.getFullPath(name)
	data, err := y.client.getFile(fullname)
	if err != nil {
		return []byte{}, &fs.PathError{Op: "read", Path: y.path, Err: err}
	}
	return data, nil
}

// ReadDir implements fs.ReadDirFS
func (y *ydfs) ReadDir(name string) ([]fs.DirEntry, error) {
	fullname := y.getFullPath(name)
	res, err := y.client.getResourceWithEmbedded(fullname)
	if err != nil {
		return []fs.DirEntry{}, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	if res.Type != "dir" {
		return []fs.DirEntry{}, &fs.PathError{Op: "readdirent", Path: name, Err: fmt.Errorf("not a directory")}
	}
	entries := make([]fs.DirEntry, len(res.Embedded.Items))

	// TODO: implement sort by filename
	for i := range res.Embedded.Items {
		entries[i] = &ydinfo{res.Embedded.Items[i]}
	}
	return entries, nil
}

func (y *ydfs) WriteFile(name string, data []byte) error {
	fullname := y.getFullPath(name)
	if err := y.client.putFileTruncate(fullname, data); err != nil {
		return &fs.PathError{Op: "write", Path: name, Err: err}
	}
	return nil
}

func (y *ydfs) Mkdir(name string) error {
	fullname := y.getFullPath(name)
	if err := y.client.mkdir(fullname); err != nil {
		return &fs.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return nil
}

func (y *ydfs) MkdirAll(dir string) error {
	fullname := y.getFullPath(dir)
	split := strings.Split(strings.Trim(fullname, "/"), "/")
	toMake := bytes.Buffer{}
	for i := range split {
		toMake.WriteString("/" + split[i])
		s, err := y.Stat(toMake.String())
		if err != nil && !errors.Is(err, ErrNotFound) {
			return &fs.PathError{Op: "mkdir", Path: y.trimRootPath(toMake.String()), Err: err}
		} else if err == nil && !s.IsDir() {
			return &fs.PathError{Op: "mkdir", Path: y.trimRootPath(toMake.String()), Err: fmt.Errorf("not a directory")}
		} else if err == nil && s.IsDir() {
			continue
		}
		if err := y.Mkdir(toMake.String()); err != nil {
			return err
		}
	}
	return nil
}

// Remove implements FS
func (y *ydfs) Remove(name string) error {
	fullname := y.getFullPath(name)
	res, err := y.client.getResourceWithEmbedded(fullname)
	if err != nil {
		return &fs.PathError{Op: "stat", Path: name, Err: err}
	} else if res.Type == "dir" && len(res.Embedded.Items) > 0 {
		return &fs.PathError{Op: "remove", Path: name, Err: fmt.Errorf("directory not empty")}
	}
	if err := y.client.delResourcePermanently(name); err != nil {
		return &fs.PathError{Op: "remove", Path: name, Err: err}
	}
	return nil
}

// RemoveAll implements FS
func (y *ydfs) RemoveAll(dir string) error {
	fullname := y.getFullPath(dir)
	res, err := y.client.getResourceWithEmbedded(fullname)
	if err != nil && errors.Is(err, ErrNotFound) {
		return nil
	} else if err != nil {
		return &fs.PathError{Op: "remove", Path: dir, Err: err}
	}
	// remove children first
	for i := range res.Embedded.Items {
		if err := y.RemoveAll(res.Embedded.Items[i].Path); err != nil {
			return err
		}
	}
	// remove parent
	if err := y.client.delResourcePermanently(dir); err != nil {
		return &fs.PathError{Op: "remove", Path: dir, Err: err}
	}
	return nil
}

func normalizeResource(r *resource) {
	r.Path = strings.Replace(r.Path, "disk:", "", 1)
	if r.Path == "/" && r.Name == "disk" {
		r.Name = "/"
	}
}
