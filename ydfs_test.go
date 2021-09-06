package ydfs

import (
	"bytes"
	"io/fs"
	"os"
	"testing"
)

var (
	testFileBody = []byte("this is a test file")
	testFileName = "/test.txt"
	testDirName  = "/test/"
)

func TestWriteFile(t *testing.T) {
	fsys, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	if err := fsys.WriteFile(testFileName, testFileBody); err != nil {
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
	t.Logf("n = %v, err = %v\ndata: %v", n, err, string(buf))
	stats, err := file.Stat()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(buf, testFileBody) {
		t.Errorf("test file received from disk differs")
	}
	t.Log(stats.Name(), stats.Size(), stats.IsDir())
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
	t.Logf("%+v", stat)
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
	for _, entry := range entries {
		t.Logf("name: %v, is dir: %v", entry.Name(), entry.IsDir())
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
