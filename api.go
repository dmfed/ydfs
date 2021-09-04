package ydfs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const (
	// base URL
	urlBase string = "https://cloud-api.yandex.net/v1/disk"

	// URLs for resource manipulations
	urlResources             = urlBase + "/resources"          // get resources metainfo
	urlResourcesDownload     = urlResources + "/download"      // download resources
	urlResourcesUpload       = urlResources + "/upload"        // upload resources
	urlResourcesPublish      = urlResources + "/publish"       // publish resources
	urlResourcesCopy         = urlResources + "/copy"          // copy resources
	urlResourcesMove         = urlResources + "/move"          // move resources
	urlResourcesFiles        = urlResources + "/files"         // list files sorted alphabetically
	urlResourcesLastUploaded = urlResources + "/last-uploaded" // list files by upload date
	urlResourcesPublic       = urlResources + "/public"        // list published files

	// URLs for manipulations with public resources
	urlPublicResources           = urlBase + "/public/resources"
	urlPublicResourcesDownload   = urlPublicResources + "/download"
	urlPublicResourcesSaveToDisk = urlPublicResources + "/save-to-disk"

	// URLs for manipulations with trashed resources
	urlTrashResources        = urlBase + "/trash/resources"
	urlTrashResourcesRestore = urlTrashResources + "/restore"

	// Async operations status
	urlOperations = urlBase + "/operations"
)

type apiclient struct {
	header http.Header
	client *http.Client
}

func newApiClient(token string, c *http.Client) *apiclient {
	h := make(http.Header)
	h.Add("Authorization", "OAuth "+token)
	h.Add("Accept", "application/json")
	h.Add("Content-Type", "application/json")
	return &apiclient{header: h, client: c}
}

// processes request returns response body bytes and error
// if we're getting non-OK status the method tries to unmarshal
// api error and return it.
func (c *apiclient) do(r *http.Request) ([]byte, error) {
	r.Header = c.header
	var (
		resp *http.Response
		err  error
	)
	resp, err = c.client.Do(r)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var e errAPI
		if err = json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		err = e
		return nil, err
	}

	return data, err
}

func (c *apiclient) getFile(ctx context.Context, name string) ([]byte, error) {
	u, err := url.Parse(urlResourcesDownload)
	if err != nil {
		return nil, err
	}
	v := make(url.Values)
	v.Add("path", name)
	u.RawQuery = v.Encode()

	var l = &link{}
	if err = c.doRequest(ctx, http.MethodGet, u, nil, l); err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, l.Href, nil)
	if err != nil {
		return nil, err
	}
	return c.do(r)
}

func (c *apiclient) getSingleResource(ctx context.Context, name string) (r resource, err error) {
	u, err := url.Parse(urlResources)
	if err != nil {
		return
	}
	v := make(url.Values)
	v.Add("path", name)
	v.Add("limit", "0")
	u.RawQuery = v.Encode()
	err = c.doRequest(ctx, http.MethodGet, u, nil, &r)
	return
}

func (c *apiclient) isOperational() bool {
	var d = &diskInfo{}
	u, _ := url.Parse(urlBase)
	err := c.doRequest(context.TODO(), http.MethodGet, u, nil, d)
	if err != nil {
		return false
	}
	return true
}

// doRequest performs most of the weight lifting with API. If result argument it non-nil
// then the method tries to unmarshal response into the passed interface.
// If no body is expected in response or the body needs to be thrown away,
// result must be nil.
func (c *apiclient) doRequest(ctx context.Context, method string, u *url.URL, body io.Reader, result interface{}) (err error) {
	var r *http.Request
	r, err = http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return
	}

	// If nil result argument is passed, we don't want
	// the resp body unmarshalled. returning.
	if result == nil {
		return
	}

	var data []byte
	if data, err = c.do(r); err != nil {
		return
	}

	// If non-nil result argument is passed we'll try to
	// unmarshal resp body into the interface provided.
	err = json.Unmarshal(data, &result)

	return
}
