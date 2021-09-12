package ydfs

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
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
	// Open
	Open(name string) (fs.File, error)
	Stat(name string) (fs.FileInfo, error)
	// Sub(dir string) (FS, error)
	ReadFile(name string) ([]byte, error)

	// TODO: comment
	ReadDir(name string) ([]fs.DirEntry, error)

	// WriteFile writes data to the named file, creating it if necessary.
	// If the file does not exist, WriteFile creates it
	// otherwise WriteFile truncates it before writing.
	WriteFile(name string, data []byte) error

	// Mkdir creates a new directory in a Disk with the specified name
	Mkdir(name string) error
	// MkdirAll creates a directory named path, along with any necessary parents,
	// and returns nil, or else returns an error.
	// MkdirAll(path string) error

	// Remove removes the named file or (empty) directory.
	Remove(name string) error
	// RemoveAll removes path and any children it contains. It removes everything it can
	// but returns the first error it encounters. If the path does not exist,
	// RemoveAll returns nil (no error).
	RemoveAll(path string) error
}

// ydfs implements FS interface
type ydfs struct {
	client *apiclient
	path   string
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
	return &ydfs{client: c, path: "/"}, nil
}

// Open implements fs.Fs interface
func (f *ydfs) Open(name string) (fs.File, error) {
	res, err := f.client.getResourceSingle(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	normalizeResource(&res)
	var file ydfile
	file.client = f.client
	file.path = res.Path
	file.isdir = (res.Type == "dir")
	return &file, nil
}

// Stat implements fs.StatFS
func (f *ydfs) Stat(name string) (fs.FileInfo, error) {
	res, err := f.client.getResourceSingle(name)
	if err != nil {
		return nil, err
	}
	normalizeResource(&res)
	return &ydinfo{res}, nil
}

// Sub implements fs.SubFS
func (f *ydfs) Sub(dir string) (FS, error) {
	res, err := f.client.getResourceSingle(dir)
	if err != nil {
		return nil, err
	}
	if res.Type != "dir" {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	return &ydfs{client: f.client, path: dir}, nil
}

// ReadFile implements fs.ReadFileFS
func (f *ydfs) ReadFile(name string) ([]byte, error) {
	return f.client.getFile(name)
}

// ReadDir implements fs.ReadDirFS
func (f *ydfs) ReadDir(name string) ([]fs.DirEntry, error) {
	res, err := f.client.getResourceWithEmbedded(name)
	if err != nil {
		return []fs.DirEntry{}, err
	}
	entries := make([]fs.DirEntry, len(res.Embedded.Items))
	for i, r := range res.Embedded.Items {
		entries[i] = &ydinfo{r}
	}
	return entries, nil
}

func (f *ydfs) WriteFile(name string, data []byte) error {
	return f.client.putFileTruncate(name, data)
}

func (f *ydfs) Mkdir(name string) error {
	return f.client.mkdir(name)
}

/* func (f *ydfs) MkdirAll(path string) error {
	split := strings.Split(strings.Trim(path, "/"))
	start := 0
	toMake := ""
	for _, dirname := range split {
		toMake += "/" + dirname
		s, err := f.Stat(toMake)
		if err != nil {

		}
	}
} */

// Remove
func (f *ydfs) Remove(name string) error {
	return f.client.delResourcePermanently(name)
}

// TODO
func (f *ydfs) RemoveAll(path string) error {
	return nil
}

// ydfile implements File interface
type ydfile struct {
	client   *apiclient
	path     string
	isdir    bool
	rdoffset int
	roffset  int
	data     []byte
}

// Read implements fs.File
func (f *ydfile) Read(b []byte) (int, error) {
	if f.data == nil {
		fileBytes, err := f.client.getFile(f.path)
		if err != nil {
			return 0, err
		}
		f.data = fileBytes
		f.roffset = 0
	}
	if f.roffset == len(f.data) {
		return 0, io.EOF
	}
	var err error
	toRead := len(b)
	if toRead > len(f.data[f.roffset:]) {
		toRead = len(f.data[f.roffset:])
		err = io.EOF
	}
	for i := 0; i < toRead; i++ {
		b[i] = f.data[f.roffset]
		f.roffset++
	}
	return toRead, err
}

// Stat implements fs.File.
func (f *ydfile) Stat() (fs.FileInfo, error) {
	res, err := f.client.getResourceSingle(f.path)
	if err != nil {
		return nil, err
	}
	normalizeResource(&res)
	return &ydinfo{res}, err
}

// Close implements fs.File
func (f *ydfile) Close() error {
	f.roffset = len(f.data)
	return nil
}

// ReadDir implements fs.ReadDirFile.
func (f *ydfile) ReadDir(n int) ([]fs.DirEntry, error) {
	res, err := f.client.getResourceWithEmbedded(f.path)
	if err != nil {
		return []fs.DirEntry{}, err
	}
	// return err if not dir
	if res.Type != "dir" {
		// TODO: check if we should return PathError here
		return []fs.DirEntry{}, fmt.Errorf("%s is not a directory", f.path)
	}
	var (
		entries   []fs.DirEntry
		errResult error
	)
	// TODO: test logic here
	total := len(res.Embedded.Items)
	remaining := total - f.rdoffset
	if n < 1 {
		n = total
		f.rdoffset = 0
	} else if n > remaining {
		n = remaining
		errResult = io.EOF
	} else if n < remaining {
		n = remaining
	}
	entries = make([]fs.DirEntry, n)
	for i := 0; i < n; i++ {
		entries[i] = &ydinfo{res.Embedded.Items[f.rdoffset]}
		f.rdoffset++
	}
	return entries, errResult
}

// ydinfo implements fs.FileInfo and fs.DirEntry.
type ydinfo struct {
	res Resource
}

// Name implements fs.FileInfo
func (y *ydinfo) Name() string {
	return y.res.Name
}

// Size implements fs.FileInfo
func (y *ydinfo) Size() int64 {
	return y.res.Size
}

// Mode implements fs.FileInfo
func (y *ydinfo) Mode() fs.FileMode {
	if y.IsDir() {
		return 1 << (32 - 1) // the only required parameter for filemode
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

func normalizeResource(r *Resource) {
	r.Path = normalizePath(r.Path)
	r.Name = normalizeName(r.Name)
}

func normalizePath(path string) string {
	return strings.Replace(path, "disk:", "", 1)
}

func normalizeName(name string) string {
	// TODO: what if we have /disk directory?
	return strings.Replace(name, "disk", "/", 1)
}
