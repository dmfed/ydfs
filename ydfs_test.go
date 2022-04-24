package ydfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"testing"
)

var (
	testFileBody          = []byte("this is a test file")
	testRootDirName       = "/"
	testFileName          = "test.txt"
	testDirName           = "/test/"
	testCascadeDirsMake   = "/test2/test3/test4"
	testSubFSDir          = "/test2"
	testSubFSExistingDir  = "test3"
	testCascadeDirsRemove = "/test2"
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
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
		return
	}
	file, err := filesystem.Open(testFileName)
	if err != nil {
		t.Error(err)
		return
	}
	buf := make([]byte, len(testFileBody))
	n, err := file.Read(buf)
	if n != len(testFileBody) || err != nil {
		t.Errorf("file Read() fails, want n = %v, have n = %v, err: %v", len(testFileBody), n, err)
		return
	}
	stats, err := file.Stat()
	if err != nil {
		t.Error(err)
		return
	}
	if !(stats.Name() == testRootDirName+testFileName) || stats.IsDir() {
		t.Errorf("testfile Stat() method returns incorrect values, want: %v, have: %v", testFileName, stats.Name())
		return
	}
	if !bytes.Equal(buf, testFileBody) {
		t.Errorf("test file received from disk differs")
		return
	}
	anotherbuf := make([]byte, 10)
	if n, err := file.Read(anotherbuf); n > 0 || !errors.Is(err, io.EOF) {
		t.Errorf("file Read does not return EOF want n = 0, got n = %d", n)
		return
	}
	if err := file.Close(); err != nil {
		t.Errorf("file.Close() returned error: %v", err)
	}
	if _, err := file.Read(anotherbuf); err == nil {
		t.Errorf("file.Read() after file.Close() returned no error")
	}
}

func TestStatFile(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	stat, err := filesystem.Stat(testFileName)
	if err != nil {
		t.Error(err)
	}
	if stat.Name() != testRootDirName+testFileName {
		t.Errorf("Stat() for test file returns incorrect values, want: %v, have: %v", testRootDirName+testFileName, stat.Name())
	} else if stat.IsDir() {
		t.Errorf("Stat() for test file returns incorrect type, want IsDir() == false, have: %v", stat.IsDir())
	}
}

func TestStatRoot(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	stat, err := filesystem.Stat("/")
	if err != nil {
		t.Error(err)
	}
	if stat.Name() != "/" {
		t.Errorf("Stat() for root dir returns incorrect values, want: %v, have: %v", "/", stat.Name())
	} else if !stat.IsDir() {
		t.Errorf("Stat() for root dir returns incorrect type, want IsDir() == true, have: %v", stat.IsDir())
	}
	info, err := filesystem.Stat("surelynonexistententry")
	if err == nil || info != nil {
		t.Errorf("Stat nonexistent entry succeeds")
	}
}

func TestReadDirFS(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	entries, err := filesystem.ReadDir("/")
	if err != nil {
		t.Error(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name() == path.Join(testRootDirName, testFileName) && !entry.IsDir() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("dir entries did not contain test file data")
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

func TestReadOnADir(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	file, err := filesystem.Open(testDirName)
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 2048)
	if _, err := file.Read(buf); err == nil {
		t.Errorf("Read succeeds on a directory")
	}
}

func TestMkdirAll(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	if err := filesystem.MkdirAll(testCascadeDirsMake); err != nil {
		t.Error(err)
	}
}

func TestRemoveFailsOnNonEmptyDir(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
		return
	}
	if err := filesystem.Remove(testCascadeDirsRemove); err == nil {
		t.Errorf("Remove doesn't fail on non-empty dir")
	}
}

func TestSubFS(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Errorf("error creating filesystem: %v", err)
		return
	}
	subfs, err := filesystem.Sub(testSubFSDir)
	if err != nil {
		t.Errorf("Sub() returned error: %v", err)
		return
	}
	if _, err := subfs.Stat(testDirName); err == nil {
		t.Errorf("subfs Stats file below root")
		return
	}

	s, err := subfs.Stat(testSubFSExistingDir)
	if err != nil {
		t.Errorf("subfs can not stat existing file")
		return
	}
	if s.Name() != path.Join(testRootDirName, testSubFSExistingDir) {
		t.Errorf("subfs returns incorrect fileinfo")
	}
	s, err = subfs.Stat("/")
	if err != nil {
		t.Errorf("subfs can not stat its root")
		return
	}
	if s.Name() != "/" {
		t.Errorf("subfs root name is incorrect. want: %s, have: %s", "/", s.Name())
	}
}

func TestRemoveAll(t *testing.T) {
	filesystem, err := New(os.Getenv("YD"), nil)
	if err != nil {
		t.Error(err)
	}
	if err := filesystem.RemoveAll(testCascadeDirsRemove); err != nil {
		t.Errorf("error trying cascade remove: %v", err)
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
