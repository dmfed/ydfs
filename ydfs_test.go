package ydfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
)

var (
	testFileBody    = []byte("this is a test file")
	testRootDirName = "/"
	testFileName    = "test.txt"
	testDirName     = "/test/"
)

func TestWriteFile(t *testing.T) {
	fsys, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	if err := fsys.WriteFile(testRootDirName+testFileName, testFileBody); err != nil {
		t.Errorf("error writing test file: %v", err)
	}
}

func TestRead(t *testing.T) {
	fs, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	file, err := fs.Open(testFileName)
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, len(testFileBody))
	n, err := file.Read(buf)
	if n != len(testFileBody) || err != nil {
		t.Errorf("file Read() fails, want n = %v, have n = %v, err: %v", len(testFileBody), n, err)
	}
	stats, err := file.Stat()
	if err != nil {
		t.Error(err)
	}
	if !(stats.Name() == testFileName) || stats.IsDir() {
		t.Errorf("testfile Stat() method returns incorrect values, want: %v, have: %v", testFileName, stats.Name())
	}
	if !bytes.Equal(buf, testFileBody) {
		t.Errorf("test file received from disk differs")
	}
	anotherbuf := make([]byte, 10)
	if n, err := file.Read(anotherbuf); n > 0 || !errors.Is(err, io.EOF) {
		t.Errorf("file Read does not return EOF want n = 0, got n = %d", n)
	}
}

func TestIncorrectRead(t *testing.T) {
	fs, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	file, err := fs.Open("/Reading")
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 2048)
	n, err := file.Read(buf)
	t.Logf("n = %d, err = %v", n, err)
}

func TestStatRoot(t *testing.T) {
	fs, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	stat, err := fs.Stat("/")
	if err != nil {
		t.Error(err)
	}
	if stat.Name() != "/" || !stat.IsDir() {
		t.Errorf("Stat() for root dir returns incorrect values, want: %v, have: %v", "/", stat.Name())
	}
}

func TestReadDirFS(t *testing.T) {
	fs, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	entries, err := fs.ReadDir("/")
	if err != nil {
		t.Error(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name() == testFileName && !entry.IsDir() {
			found = true
		}
	}
	if !found {
		t.Errorf("direntries did not contain test file data")
		t.Logf("%+v", entries)
	}
}

func TestOpenReturnsPathErr(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	_, err = filesystem.Open("a very long unexistent name of file")
	_, ok := err.(*fs.PathError)
	if !ok {
		t.Log("Open returns incorrect error type")
	}
}

func TestMkdir(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	if err := filesystem.Mkdir(testDirName); err != nil {
		t.Errorf("error creating test dir: %v", err)
	}
	if _, err := filesystem.Stat(testDirName); err != nil {
		t.Errorf("error stat test dir: %v", err)
	}
}

func TestRemove(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	if err := filesystem.Remove(testDirName); err != nil {
		t.Errorf("error removing test dir: %v", err)
	}

	if err := filesystem.Remove(testFileName); err != nil {
		t.Errorf("error removing test file: %v", err)
	}
}
