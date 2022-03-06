package ydfs

import "io/fs"

// FS provides access to files stored in
// Yandex Disk. It complies with fs.FS, fs.GlobFS,
// fs.ReadDirFS, fs.ReadFileFS, fs.StatFS, fs.SubFS
// interfaces of standard library. FS has additional methods
// specific to metainformation stored by Yandex -
// see DiskInfo and UserInfo methods.
type FS interface {
	// CreCreate creates or truncates the named file. If the file already exists,
	// it is truncated. If the file does not exist, it is created.
	// If successful, methods on the returned File can be used for I/O.
	// If there is an error, it will be of type fs.PathError.
	Create(name string) (fs.File, error)

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
