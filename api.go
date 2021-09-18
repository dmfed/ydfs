package ydfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

var (
	ErrNetwork  = errors.New("network error")
	ErrAPI      = errors.New("API error")
	ErrNotFound = errors.New("resource not found")
	ErrUnknown  = errors.New("unknown error")
	ErrInternal = errors.New("internal error")
)

type apiclient struct {
	header http.Header
	client *http.Client
}

// newApiClient createst Yandex Disk API client, which uses
// the provided http.Client.
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
		return []byte{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	var data []byte
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	// checking if we've got one of success codes
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		var e errAPI
		if err = json.Unmarshal(data, &e); err != nil {
			return []byte{}, fmt.Errorf("%w: unknown response with code %d from API: %s", ErrUnknown, resp.StatusCode, string(data))
		}
		if e.NotFound() {
			err = fmt.Errorf("%w, %v", ErrNotFound, e)
		} else {
			err = fmt.Errorf("%w, %v", ErrAPI, e)
		}
		return []byte{}, err
	}

	return data, nil
}

// requestInterface performs most of the weight lifting with API. If result argument it non-nil
// then the method tries to unmarshal response into the passed interface.
// If no body is expected in response or the body needs to be thrown away,
// result must be nil.
func (c *apiclient) requestInterface(method string, u *url.URL, body io.Reader, result interface{}) (err error) {
	var r *http.Request
	r, err = http.NewRequest(method, u.String(), body)
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
	if err = json.Unmarshal(data, &result); err != nil {
		err = fmt.Errorf("%w: %v", ErrInternal, err)
	}
	return
}

// getDiskInfo fetches information about user's Disk.
func (c *apiclient) getDiskInfo() (info diskInfo, err error) {
	u, err := url.Parse(urlBase)
	if err != nil {
		return
	}
	err = c.requestInterface(http.MethodGet, u, nil, &info)
	return
}

// getFile fetches single file bytes.
func (c *apiclient) getFile(name string) ([]byte, error) {
	// first we need to fetch the download url
	u, err := url.Parse(urlResourcesDownload)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrInternal, err)
	}
	v := make(url.Values)
	v.Add("path", name)
	u.RawQuery = v.Encode()

	var l = &link{}
	if err = c.requestInterface(http.MethodGet, u, nil, l); err != nil {
		return []byte{}, err
	}
	if l.Templated {
		// TODO: deal with templated links (I haven't seen one yet)
	}
	// performing the actual download
	r, err := http.NewRequest(http.MethodGet, l.Href, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrInternal, err)
	}
	return c.do(r)
}

func (c *apiclient) putFile(name string, overwrite bool, data []byte) error {
	u, err := url.Parse(urlResourcesUpload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	v := make(url.Values)
	v.Add("path", name)
	if overwrite {
		v.Add("overwrite", "true")
	}
	u.RawQuery = v.Encode()
	var l = &link{}
	if err = c.requestInterface(http.MethodGet, u, nil, l); err != nil {
		return err
	}

	switch l.Templated {
	case true:
		fallthrough
	default:
	}
	// performing the actual upload
	r, err := http.NewRequest(http.MethodPut, l.Href, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	_, err = c.do(r)
	return err
}

func (c *apiclient) putFileTruncate(name string, data []byte) error {
	return c.putFile(name, true, data)
}

func (c *apiclient) putFileNoTruncate(name string, data []byte) error {
	return c.putFile(name, false, data)
}

func (c *apiclient) mkdir(name string) error {
	u, err := url.Parse(urlResources)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	v := make(url.Values)
	v.Add("path", name)
	u.RawQuery = v.Encode()
	var l = link{}
	err = c.requestInterface(http.MethodPut, u, nil, &l)
	return err
}

// getResource fetches Resource identified by name from the API.
// if limit >= 0 then len(Resource.Embedded.Items) will not exceed limit.
func (c *apiclient) getResource(name string, limit int) (r resource, err error) {
	u, err := url.Parse(urlResources)
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrInternal, err)
		return
	}
	v := make(url.Values)
	v.Add("path", name)
	if limit >= 0 {
		v.Add("limit", strconv.Itoa(limit))
	}
	u.RawQuery = v.Encode()
	err = c.requestInterface(http.MethodGet, u, nil, &r)
	return
}

// getResourceSingle fetches Resource without embedded resuorces
func (c *apiclient) getResourceSingle(name string) (resource, error) {
	return c.getResource(name, 0)
}

// getResourceWithEmbedded fetches Resource with embedded resources
func (c *apiclient) getResourceWithEmbedded(name string) (resource, error) {
	return c.getResource(name, -1)
}

func (c *apiclient) delResource(name string, permanently bool) error {
	u, err := url.Parse(urlResources)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	v := make(url.Values)
	v.Add("path", name)
	if permanently {
		v.Add("permanently", "true")
	}
	u.RawQuery = v.Encode()
	r, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	_, err = c.do(r)
	return err
}

func (c *apiclient) delResourcePermanently(name string) error {
	return c.delResource(name, true)
}

func (c *apiclient) delResourceTrash(name string) error {
	return c.delResource(name, false)
}
