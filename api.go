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

// newApiClient создаст клиент API, использующий предоставленный
// http.Client.
func newApiClient(token string, c *http.Client) *apiclient {
	h := make(http.Header)
	h.Add("Authorization", "OAuth "+token)
	h.Add("Accept", "application/json")
	h.Add("Content-Type", "application/json")
	return &apiclient{header: h, client: c}
}

// processes request returns response body bytes and error
// if we're getting non-OK status the method tries to unmarshal
// api error to struct which imlements error interface.
func (c *apiclient) do(r *http.Request) ([]byte, error) {
	r.Header = c.header
	var (
		resp *http.Response
		err  error
	)
	resp, err = c.client.Do(r)
	if err != nil {
		return []byte{}, err
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	if resp.StatusCode != http.StatusOK {
		var e errAPI
		if err = json.Unmarshal(data, &e); err != nil {
			return []byte{}, err
		}
		err = &e
		return []byte{}, err
	}

	return data, err
}

// requestInterface performs most of the weight lifting with API. If result argument it non-nil
// then the method tries to unmarshal response into the passed interface.
// If no body is expected in response or the body needs to be thrown away,
// result must be nil.
func (c *apiclient) requestInterface(ctx context.Context, method string, u *url.URL, body io.Reader, result interface{}) (err error) {
	var r *http.Request
	r, err = http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return
	}

	var data []byte
	if data, err = c.do(r); err != nil {
		return
	}
	// If nil result argument is passed, we don't want
	// the resp body unmarshalled. returning.
	if result == nil {
		return
	}
	// If non-nil result argument is passed we'll try to
	// unmarshal resp body into the interface provided.
	err = json.Unmarshal(data, &result)
	return
}

// getDiskInfo fetches information about user's Disk.
func (c *apiclient) getDiskInfo(ctx context.Context) (info DiskInfo, err error) {
	u, err := url.Parse(urlBase)
	if err != nil {
		return
	}
	err = c.requestInterface(ctx, http.MethodGet, u, nil, &info)
	return
}

// getFile fetches single file bytes.
func (c *apiclient) getFile(ctx context.Context, name string) ([]byte, error) {
	// first we need to fetch the download url
	u, err := url.Parse(urlResourcesDownload)
	if err != nil {
		return []byte{}, err
	}
	v := make(url.Values)
	v.Add("path", name)
	u.RawQuery = v.Encode()

	var l = &link{}
	if err = c.requestInterface(ctx, http.MethodGet, u, nil, l); err != nil {
		return []byte{}, err
	}
	// performing the actual download
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, l.Href, nil)
	if err != nil {
		return []byte{}, err
	}
	return c.do(r)
}

// getSingleResource fetches resource struct without embedded resources.
func (c *apiclient) getResource(ctx context.Context, name string) (r Resource, err error) {
	u, err := url.Parse(urlResources)
	if err != nil {
		return
	}
	v := make(url.Values)
	v.Add("path", name)
	v.Add("limit", "0")
	u.RawQuery = v.Encode()
	err = c.requestInterface(ctx, http.MethodGet, u, nil, &r)
	return
}

func (c *apiclient) getResourceWithEmbedded(ctx context.Context, name string) (Resource, error) {
	var r = Resource{}
	u, err := url.Parse(urlResources)
	if err != nil {
		return r, err
	}
	v := make(url.Values)
	v.Add("path", name)
	u.RawQuery = v.Encode()
	err = c.requestInterface(ctx, http.MethodGet, u, nil, &r)
	return r, err
}
