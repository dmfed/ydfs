package ydfs

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
)

var client *apiclient

func TestMain(m *testing.M) {
	token, ok := os.LookupEnv("YD")
	if !ok {
		fmt.Println("environment variable YD not set. skipping integration tests")
		os.Exit(0)
	}
	client = newApiClient(token, http.DefaultClient)
	os.Exit(m.Run())
}

func Test_doRequest(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, urlBase, nil)
	if err != nil {
		t.Errorf("error creating request %v", err)
	}
	_, err = client.do(context.TODO(), r, http.StatusOK)
	if err != nil {
		t.Errorf("client.do returned %v", err)
	}
}

func Test_requestInterface(t *testing.T) {
	var d = &diskInfo{}
	err := client.requestInterface(http.MethodGet, http.StatusOK, urlBase, nil, d)
	if err != nil {
		t.Errorf("client.requestInterface returned: %v", err)
	}
}

func Test_getResourceMinTraffic(t *testing.T) {
	res, err := client.getResourceMinTraffic("/")
	if err != nil {
		t.Errorf("client.getResourceMinTraffic returned: %v", err)
	}
	if res.Name == "" && res.Path == "" && res.Modified.IsZero() && res.Type == "" {
		t.Errorf("missing required data in response from client.getResourceMinTraffic")
	}
}

func Test_putFile(t *testing.T) {
	err := client.putFileTruncate(testFileName, testFileBody)
	if err != nil {
		t.Logf("upload test file failed: %v", err)
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
