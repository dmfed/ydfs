package ydfs

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

// ydfile implements fs.File interface, fs.ReadDirFile,
// fs.FileInfo, fs.DirEntry.
type ydfile struct {
	client   *apiclient // api client
	path     string     // file path including its name
	name     string     // file name
	isdir    bool       // sets to true if file is a directory
	modtime  time.Time  // modification time
	rdoffset int        // read dir offset for directories
	roffset  int        // read offset for regular files
	size     int64      // actual data size in bytes
	data     []byte     // payload of a file
}

func (file *ydfile) update() error {
	res, err := file.client.getResourceMinTraffic(file.path)
	if err != nil {
		return err
	}
	normalizeResource(&res)
	file.path = res.Path
	file.name = res.Name
	file.isdir = (res.Type == "dir")
	file.size = res.Size
	file.modtime = res.Modified
	return nil
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
	err := file.update()
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: file.path, Err: err}
	}
	return file, nil
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
		entries[i] = file
		file.rdoffset++
	}
	return entries, errResult
}

// Name implements fs.FileInfo
func (file *ydfile) Name() string {
	return file.name
}

// Size implements fs.FileInfo
func (file *ydfile) Size() int64 {
	return file.size
}

// Mode implements fs.FileInfo
func (file *ydfile) Mode() fs.FileMode {
	if file.IsDir() {
		return 1 << (32 - 1) // the only required parameter for filemode
	}
	return 0
}

// ModTime implements fs.FileInfo
func (file *ydfile) ModTime() time.Time {
	return file.modtime
}

// IsDir implements fs.FileInfo
func (file *ydfile) IsDir() bool {
	return file.isdir
}

// Sys implements fs.FileInfo
func (file *ydfile) Sys() interface{} {
	return nil
}

// Type implements fs.DirEntry
func (file *ydfile) Type() fs.FileMode {
	return file.Mode()
}

// Info implements fs.DirEntry
func (file *ydfile) Info() (fs.FileInfo, error) {
	return file, nil
}
