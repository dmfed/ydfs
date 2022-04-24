package ydfs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// FS provides access to files stored in
// Yandex Disk. It complies with fs.FS, fs.GlobFS,
// fs.ReadDirFS, fs.ReadFileFS, fs.StatFS, fs.SubFS
// interfaces of standard library. FS has additional methods
// specific to metainformation stored by Yandex -
// see DiskInfo and UserInfo methods.
type FS interface {
	// Open opens the named file.
	Open(name string) (fs.File, error)

	// Stat returns a FileInfo describing the named file from the file system.
	Stat(name string) (fs.FileInfo, error)

	// Sub returns an FS corresponding to the subtree rooted at dir.
	Sub(dir string) (FS, error)

	// ReadFile reads the named file and returns its contents.
	// A successful call returns a nil error, not io.EOF.
	// (Because ReadFile reads the whole file, the expected EOF
	// from the final Read is not treated as an error to be reported.)
	//
	// The caller is permitted to modify the returned byte slice.
	// This method should return a copy of the underlying data.
	ReadFile(name string) ([]byte, error)

	// ReadDir reads the named directory
	// and returns a list of directory entries sorted by filename.
	ReadDir(name string) ([]fs.DirEntry, error)

	// WriteFile writes data to the named file, creating it if necessary.
	// If the file does not exist, WriteFile creates it
	// otherwise WriteFile truncates it before writing.
	WriteFile(name string, data []byte) error

	// Mkdir creates a new directory with the specified name
	Mkdir(name string) error

	// MkdirAll creates a directory named path, along with any necessary parents,
	// and returns nil, or else returns an error.
	MkdirAll(path string) error

	// Remove removes the named file or (empty) directory.
	Remove(name string) error

	// RemoveAll removes path and any children it contains. It removes everything it can
	// but returns the first error it encounters. If the path does not exist,
	// RemoveAll returns nil (no error).
	RemoveAll(path string) error
}

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

// Open implements fs.Fs interface
func (y *ydfs) Open(name string) (fs.File, error) {
	var fullname string
	if y.issub {
		fullname = path.Join(y.path, name)
	} else {
		fullname = name
	}
	res, err := y.client.getResourceMinTraffic(fullname)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	normalizeResourcePath(&res)
	var file ydfile
	file.client = y.client
	file.path = res.Path
	file.isdir = (res.Type == "dir")
	file.size = res.Size
	return &file, nil
}

// Stat implements fs.StatFS
func (y *ydfs) Stat(name string) (fs.FileInfo, error) {
	if y.issub {
		name = path.Join(y.path, name)
	}
	res, err := y.client.getResourceMinTraffic(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: y.path, Err: err}
	}
	normalizeResourcePath(&res)
	if y.issub {
		res.Path = strings.TrimPrefix(res.Path, y.path)
		if res.Path == "" {
			res.Path = "/"
		}
	}
	return &ydinfo{res}, nil
}

// Sub implements fs.SubFS
func (y *ydfs) Sub(dir string) (FS, error) {
	if y.issub {
		dir = path.Join(y.path, dir)
	}
	res, err := y.client.getResourceMinTraffic(dir)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: y.path, Err: err}
	}
	if res.Type != "dir" {
		return nil, &fs.PathError{Op: "sub", Path: y.path, Err: fmt.Errorf("not a directory")}
	}
	normalizeResourcePath(&res)
	return &ydfs{client: y.client, path: res.Path, issub: true}, nil
}

// ReadFile implements fs.ReadFileFS
func (y *ydfs) ReadFile(name string) ([]byte, error) {
	if y.issub {
		name = path.Join(y.path, name)
	}
	data, err := y.client.getFile(name)
	if err != nil {
		return []byte{}, &fs.PathError{Op: "read", Path: y.path, Err: err}
	}
	return data, nil
}

// ReadDir implements fs.ReadDirFS
func (y *ydfs) ReadDir(name string) ([]fs.DirEntry, error) {
	if y.issub {
		name = path.Join(y.path, name)
	}
	res, err := y.client.getResourceWithEmbedded(name)
	if err != nil {
		return []fs.DirEntry{}, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	if res.Type != "dir" {
		return []fs.DirEntry{}, &fs.PathError{Op: "readdirent", Path: name, Err: fmt.Errorf("not a directory")}
	}
	entries := make([]fs.DirEntry, len(res.Embedded.Items))

	// TODO: implement sort by filename
	for i := 0; i < len(res.Embedded.Items); i++ {
		entries[i] = &ydinfo{res.Embedded.Items[i]}
	}
	return entries, nil
}

func (y *ydfs) WriteFile(name string, data []byte) error {
	if y.issub {
		name = path.Join(y.path, name)
	}
	if err := y.client.putFileTruncate(name, data); err != nil {
		return &fs.PathError{Op: "write", Path: name, Err: err}
	}
	return nil
}

func (y *ydfs) Mkdir(name string) error {
	if y.issub {
		name = path.Join(y.path, name)
	}
	if err := y.client.mkdir(name); err != nil {
		return &fs.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return nil
}

func (y *ydfs) MkdirAll(dir string) error {
	if y.issub {
		dir = path.Join(y.path, dir)
	}
	split := strings.Split(strings.Trim(dir, "/"), "/")
	toMake := bytes.Buffer{}
	for i := range split {
		toMake.WriteString("/" + split[i])
		s, err := y.Stat(toMake.String())
		if err != nil && !errors.Is(err, ErrNotFound) {
			return &fs.PathError{Op: "mkdir", Path: toMake.String(), Err: err}
		} else if err == nil && !s.IsDir() {
			return &fs.PathError{Op: "mkdir", Path: toMake.String(), Err: fmt.Errorf("not a directory")}
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
	if y.issub {
		name = path.Join(y.path, name)
	}
	res, err := y.client.getResourceWithEmbedded(name)
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
	if y.issub {
		dir = path.Join(y.path, dir)
	}
	res, err := y.client.getResourceWithEmbedded(dir)
	if err != nil && errors.Is(err, ErrNotFound) {
		return nil
	} else if err != nil {
		return &fs.PathError{Op: "remove", Path: dir, Err: err}
	}
	// remove children first
	for i := 0; i < len(res.Embedded.Items); i++ {
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

// ydfile implements File interface
type ydfile struct {
	client *apiclient // api client
	path   string     // file path including its name
	// name     string     // file name
	isdir bool // sets to true if file is a directory
	// mode     fs.FileMode
	rdoffset int    // read dir offset for directories
	roffset  int    // read offset for regular files
	size     int64  // actual data size in bytes
	data     []byte // payload of a file
}

// Read implements fs.File
func (file *ydfile) Read(b []byte) (int, error) {
	if file.isdir {
		return 0, &fs.PathError{Op: "read", Path: file.path, Err: fmt.Errorf("is a directory")}
	}
	// TODO: implement download in chunks to only fetch
	// required data
	if file.data == nil {
		fileBytes, err := file.client.getFile(file.path)
		if err != nil {
			return 0, &fs.PathError{Op: "read", Path: file.path, Err: err}
		}
		file.data = fileBytes
		file.roffset = 0
	}
	if file.roffset == len(file.data) {
		return 0, io.EOF
	}
	var (
		err         error
		toRead      int = len(b)
		doneReading int = 0
	)
	if toRead > len(file.data[file.roffset:]) {
		toRead = len(file.data[file.roffset:])
		err = io.EOF
	}
	for i := 0; i < toRead; i++ {
		b[i] = file.data[file.roffset]
		file.roffset++
		doneReading++
	}
	return doneReading, err
}

// Stat implements fs.File.
func (file *ydfile) Stat() (fs.FileInfo, error) {
	res, err := file.client.getResourceMinTraffic(file.path)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: file.path, Err: err}
	}
	normalizeResourcePath(&res)
	return &ydinfo{res}, err
}

// Close implements fs.File
func (file *ydfile) Close() error {
	file.data = []byte{}
	file.roffset = 0
	return nil
}

// ReadDir implements fs.ReadDirFile.
func (file *ydfile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !file.isdir {
		return []fs.DirEntry{}, &fs.PathError{Op: "readdirent", Path: file.path, Err: fmt.Errorf("not a directory")}
	}
	res, err := file.client.getResourceWithEmbedded(file.path)
	if err != nil {
		return []fs.DirEntry{}, &fs.PathError{Op: "readdirent", Path: file.path, Err: err}
	}
	var (
		entries   []fs.DirEntry
		errResult error
	)
	// TODO: test logic here
	total := len(res.Embedded.Items)
	remaining := total - file.rdoffset
	if n < 1 {
		n = total
		file.rdoffset = 0
	} else if n > remaining {
		n = remaining
		errResult = io.EOF
	} else if n < remaining {
		n = remaining
	}
	entries = make([]fs.DirEntry, n)
	for i := 0; i < n; i++ {
		entries[i] = &ydinfo{res.Embedded.Items[file.rdoffset]}
		file.rdoffset++
	}
	return entries, errResult
}

// ydinfo implements fs.FileInfo and fs.DirEntry.
type ydinfo struct {
	res resource
}

// Name implements fs.FileInfo
func (y *ydinfo) Name() string {
	normalizeResourcePath(&y.res)
	return y.res.Path
}

// Size implements fs.FileInfo
func (y *ydinfo) Size() int64 {
	return y.res.Size
}

// Mode implements fs.FileInfo
func (y *ydinfo) Mode() fs.FileMode {
	if y.IsDir() {
		return fs.ModeDir
	}
	return 0
}

// ModTime implements fs.FileInfo
func (y *ydinfo) ModTime() time.Time {
	return y.res.Modified
}

// IsDir implements fs.FileInfo
func (y *ydinfo) IsDir() bool {
	return y.res.Type == "dir"
}

// Sys implements fs.FileInfo
func (y *ydinfo) Sys() interface{} {
	return nil
}

// Type implements fs.DirEntry
func (y *ydinfo) Type() fs.FileMode {
	return y.Mode()
}

// Info implements fs.DirEntry
func (y *ydinfo) Info() (fs.FileInfo, error) {
	return y, nil
}

func normalizeResourcePath(r *resource) {
	r.Path = strings.Replace(r.Path, "disk:", "", 1)
	if r.Path == "/" && r.Name == "disk" {
		r.Name = "/"
	}
}
