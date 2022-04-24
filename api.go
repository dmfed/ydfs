package ydfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

var minimalFields = []string{"name", "path", "type", "size", "modified"}

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
// if we're getting status not equal to the requiredcode the method tries to unmarshal
// response to errAPI struct which imlements error interface.
func (c *apiclient) do(ctx context.Context, r *http.Request, requiredcode int) ([]byte, error) {
	r.Header = c.header
	var (
		resp *http.Response
		err  error
		data []byte
	)
	if ctx != nil {
		r = r.WithContext(ctx)
	}
	resp, err = c.client.Do(r)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	// checking if we've got correct result code
	if resp.StatusCode != requiredcode {
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

// requestInterface performs some of the weight lifting with API. If result argument it non-nil
// then the method tries to unmarshal response into the passed interface.
// If no body is expected in response or the body needs to be thrown away,
// result must be nil.
func (c *apiclient) requestInterface(method string, respcode int, url string, body io.Reader, result interface{}) (err error) {
	var (
		r    *http.Request
		data []byte
	)
	r, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	if data, err = c.do(context.TODO(), r, respcode); err != nil {
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
	err = c.requestInterface(http.MethodGet, http.StatusOK, urlBase, nil, &info)
	return
}

// getFile fetches single file bytes.
func (c *apiclient) getFile(name string) ([]byte, error) {
	// first we need to fetch the download url

	v := make(url.Values)
	v.Add("path", name)
	url := urlResourcesDownload + "?" + v.Encode()
	var l = &link{}
	if err := c.requestInterface(http.MethodGet, http.StatusOK, url, nil, l); err != nil {
		return []byte{}, err
	}
	if l.Templated {
		// TODO: deal with templated links (I haven't seen one yet)
	}
	// performing the actual download
	r, err := http.NewRequest(l.Method, l.Href, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("%w: %v", ErrInternal, err)
	}
	return c.do(context.TODO(), r, http.StatusOK)
}

func (c *apiclient) putFile(name string, overwrite bool, data []byte) error {
	v := make(url.Values)
	v.Add("path", name)
	if overwrite {
		v.Add("overwrite", "true")
	}

	url := urlResourcesUpload + "?" + v.Encode()
	var l = &link{}
	if err := c.requestInterface(http.MethodGet, http.StatusOK, url, nil, l); err != nil {
		return err
	}

	if l.Templated {
		// TODO: deal with templated links (I haven't seen one yet)
	}

	// performing the actual upload
	r, err := http.NewRequest(l.Method, l.Href, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	_, err = c.do(context.TODO(), r, http.StatusCreated)
	return err
}

func (c *apiclient) putFileTruncate(name string, data []byte) error {
	return c.putFile(name, true, data)
}

func (c *apiclient) putFileNoTruncate(name string, data []byte) error {
	return c.putFile(name, false, data)
}

func (c *apiclient) mkdir(name string) error {
	v := make(url.Values)
	v.Add("path", name)
	url := urlResources + "?" + v.Encode()
	var l = link{}
	return c.requestInterface(http.MethodPut, http.StatusCreated, url, nil, &l)
}

// getResource fetches Resource identified by name from the API.
// if limit == 0 then embedded resources will not be requested not included
// if limit > 0 then len(Resource.Embedded.Items) will not exceed limit.
func (c *apiclient) getResource(name string, limit int, fields ...string) (r resource, err error) {
	v := make(url.Values)
	v.Add("path", name)
	v.Add("limit", strconv.Itoa(limit))
	if len(fields) > 0 {
		v.Add("fields", strings.Join(fields, ","))
	}
	url := urlResources + "?" + v.Encode()
	fmt.Printf("URL: %v\n", url)
	err = c.requestInterface(http.MethodGet, http.StatusOK, url, nil, &r)
	return
}

// getResourceSingle fetches resource without embedded resources
func (c *apiclient) getResourceSingle(name string) (resource, error) {
	return c.getResource(name, 0)
}

// getResourceMinTraffic fetches resource only requesting minimum
// required onfo for FS to fucntion. minimalFields is globally declared.
func (c *apiclient) getResourceMinTraffic(name string) (resource, error) {
	return c.getResource(name, 0, minimalFields...)
}

// getResourceWithEmbedded fetches resource with embedded resources
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
		return err
	}
	_, err = c.do(context.TODO(), r, http.StatusNoContent)
	return err
}

func (c *apiclient) delResourcePermanently(name string) error {
	return c.delResource(name, true)
}

func (c *apiclient) delResourceTrash(name string) error {
	return c.delResource(name, false)
}
