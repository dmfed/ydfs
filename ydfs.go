package ydfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"
)

var ctx = context.TODO()

// FS provides access to files stored in
// Yandex Disk. It implements fs.FS interface of
// standard library. FS has additional methods
// specific to metainformation stored by Yandex -
// see DiskInfo and UserInfo methods.
type FS struct {
	client *apiclient
	path   string
}

// New returns ydfs.FS which is compliant with
// standard library's fs.FS interface. Token is required for authorization.
// If client is nil then http.DefaultClient is used.
func New(token string, client *http.Client) (fs.FS, error) {
	if client == nil {
		client = http.DefaultClient
	}
	c := newApiClient(token, client)
	// checking whether we can fetch disk metadata to
	// make sure that token is valid and we we can send
	// requests to the API.
	if _, err := c.getDiskInfo(ctx); err != nil {
		return nil, err
	}
	return &FS{client: c, path: "/"}, nil
}

// Open implements fs.Fs interface
func (f *FS) Open(name string) (fs.File, error) {
	res, err := f.client.getResource(ctx, name)
	if err != nil {
		return nil, err
	}
	file := File{f.client, name, res, nil}
	return &file, nil
}

// Stat implements fs.StatFS
func (f *FS) Stat(name string) (fs.FileInfo, error) {
	res, err := f.client.getResource(ctx, name)
	if err != nil {
		return nil, err
	}
	return &FileInfo{res}, nil
}

// Sub implements fs.SubFS
func (f *FS) Sub(dir string) (fs.FS, error) {
	res, err := f.client.getResource(ctx, dir)
	if err != nil {
		return nil, err
	}
	if res.Type != "dir" {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	return &FS{client: f.client, path: dir}, nil
}

// ReadFile implements fs.ReadFileFS
func (f *FS) ReadFile(name string) ([]byte, error) {
	return f.client.getFile(ctx, name)
}

// ReadDir implements fs.ReadDirFS
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	res, err := f.client.getResourceWithEmbedded(ctx, name)
	if err != nil {
		return []fs.DirEntry{}, err
	}
	entries := make([]fs.DirEntry, len(res.Embedded.Items))
	for i, r := range res.Embedded.Items {
		entries[i] = &FileInfo{r}
	}
	return entries, nil
}

// Below are Specific methods of FS which fetch info from
// Yandex Disk API.

// DiskInfo fetches current metadata of a disk.
// If fetched successfully, returns non-empty DiskInfo.
// Always returns non-nil struct, which is empty in case of
// error.
func (f *FS) DiskInfo() (DiskInfo, error) {
	return f.client.getDiskInfo(ctx)
}

// UserInfo fetches current user details.
// If fetched successfully, returns non-empty User struct.
// Always returns non-nil struct, which is empty in case of
// error.
func (f *FS) UserInfo() (User, error) {
	info, err := f.client.getDiskInfo(ctx)
	if err != nil {
		return User{}, err
	}
	return info.User, nil
}

// A File provides access to a single file and
// represents a file stored in a cloud. It is fully compliant
// with fs.File interface.
type File struct {
	client *apiclient
	path   string
	res    Resource
	data   []byte
}

// Read implements fs.File
func (f *File) Read(b []byte) (int, error) {
	fileBytes, err := f.client.getFile(ctx, f.path)
	if err != nil {
		return 0, err
	}
	f.data = fileBytes
	rdr := bytes.NewReader(f.data)
	return rdr.Read(b)
}

// Stat implements fs.File.
func (f *File) Stat() (fs.FileInfo, error) {
	res, err := f.client.getResource(ctx, f.path)
	if err != nil {
		return nil, err
	}
	return &FileInfo{res}, err
}

// Close implements fs.File
func (f *File) Close() error {
	return nil
}

// ReadDir implements fs.ReadDirFile.
func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	// return err if not dir
	res, err := f.client.getResourceWithEmbedded(ctx, f.path)
	if err != nil {
		return []fs.DirEntry{}, err
	}
	if res.Type != "dir" {
		return []fs.DirEntry{}, fmt.Errorf("%s is not a directory", f.path)
	}
	var (
		entries   []fs.DirEntry
		errResult error
	)
	if n <= 0 {
		entries = make([]fs.DirEntry, len(res.Embedded.Items))
		n = len(res.Embedded.Items)
		errResult = nil
	} else if n > len(res.Embedded.Items) {
		errResult = io.EOF
		entries = make([]fs.DirEntry, n)
	}
	for i := 0; i < n; i++ {
		entries[i] = &FileInfo{res.Embedded.Items[i]}
	}
	return entries, errResult
}

// FileInfo describes a file and is returned by Stat.
type FileInfo struct {
	res Resource
}

// Name implements fs.FileInfo
func (f *FileInfo) Name() string {
	return f.res.Name
}

// Size implements fs.FileInfo
func (f *FileInfo) Size() int64 {
	return f.res.Size
}

// Mode implements fs.FileInfo
func (f *FileInfo) Mode() fs.FileMode {
	if f.IsDir() {
		return 1 << (32 - 1)
	}
	return 0
}

// ModTime implements fs.FileInfo
func (f *FileInfo) ModTime() time.Time {
	return f.res.Modified
}

// IsDir implements fs.FileInfo
func (f *FileInfo) IsDir() bool {
	return f.res.Type == "dir" || false
}

// Sys implements fs.FileInfo
func (f *FileInfo) Sys() interface{} {
	return nil
}

// Type implements fs.DirEntry
func (f *FileInfo) Type() fs.FileMode {
	return f.Mode()
}

// Info implements fs.DirEntry
func (f *FileInfo) Info() (fs.FileInfo, error) {
	return f, nil
}
