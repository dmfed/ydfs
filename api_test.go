package ydfs

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"testing"
)

var client = newApiClient(os.Getenv("YD"), http.DefaultClient)

func Test_requestInterface(t *testing.T) {
	var d = &DiskInfo{}
	u, _ := url.Parse(urlBase)
	err := client.requestInterface(http.MethodGet, u, nil, d)
	if err != nil {
		t.Errorf("error with correct credentials: %v", err)
	}
	t.Log(d)
}

func Test_putFile(t *testing.T) {
	err := client.putFileTruncate(testFileName, testFileBody)
	if err != nil {
		t.Logf("upload failed: %v", err)
	}
}

func Test_getFile(t *testing.T) {
	b, err := client.getFile(testFileName)
	if err != nil {
		t.Errorf("getting test file failed: %v", err)
	}

	if !bytes.Equal(b, testFileBody) {
		t.Errorf("error comparing testfile with fetched result")
	}
}

func Test_GetPaths(t *testing.T) {
	for _, path := range []string{"/", "/go.mod", "/nulls10.b", "/Reading/Math"} {
		res, err := client.getResourceSingle(path)
		if err != nil {
			t.Error(err)
		}
		t.Logf("name: %s, path: %s", res.Name, res.Path)
	}
	root, _ := os.Stat("/")
	t.Logf("root name: %s", root.Name())
}
