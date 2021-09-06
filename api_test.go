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
