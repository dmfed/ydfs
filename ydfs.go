package ydfs

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"time"
)

type ydfs struct {
	client *apiclient
	path   string
}

func (y *ydfs) Open(name string) (fs.File, error) {
	if _, err := y.client.getSingleResource(context.TODO(), name); err != nil {
		return nil, err
	}
	f := ydfs{y.client, name}
	return &f, nil
}

func (y *ydfs) Close() error {
	return nil
}

func (y *ydfs) Read(b []byte) (int, error) {
	fileBytes, err := y.client.getFile(context.TODO(), y.path)
	if err != nil {
		return 0, err
	}
	rdr := bytes.NewReader(fileBytes)
	return rdr.Read(b)
}

func (y *ydfs) Stat() (fs.FileInfo, error) {
	res, err := y.client.getSingleResource(context.TODO(), y.path)
	if err != nil {
		return nil, err
	}
	return &ydinfo{res}, err
}

type ydinfo struct {
	res resource
}

func (r *ydinfo) Name() string {
	return r.res.Name
}

func (r *ydinfo) Size() int64 {
	return r.res.Size
}

func (r *ydinfo) Mode() fs.FileMode {
	if r.IsDir() {
		return 1 << (32 - 1)
	}
	return 0
}

func (r *ydinfo) ModTime() time.Time {
	return r.res.Modified
}

func (r *ydinfo) IsDir() bool {
	return r.res.Type == "dir" || false
}

func (r *ydinfo) Sys() interface{} {
	return nil
}

func New(token string, client *http.Client) (fs.FS, error) {
	if client == nil {
		client = http.DefaultClient
	}
	c := newApiClient(token, client)
	return &ydfs{c, "/"}, nil
}
