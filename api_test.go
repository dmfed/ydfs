package ydfs

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
)

var client = newApiClient(os.Getenv("YD"), http.DefaultClient)

var ctx = context.Background()

func Test_doRequest(t *testing.T) {
	var d = &diskInfo{}
	u, _ := url.Parse(urlBase)
	err := client.doRequest(ctx, http.MethodGet, u, nil, d)
	if err != nil {
		t.Errorf("error with correct credentials: %v", err)
	}
	t.Logf("%+v", d)
}

func Test_getSingleResource(t *testing.T) {
	res, err := client.getSingleResource(ctx, "/")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", res)
}

func Test_getFile(t *testing.T) {
	b, err := client.getFile(ctx, "/go.mod")
	if err != nil {
		t.Error(err)
	}
	t.Log(string(b))
}
