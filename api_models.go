package ydfs

import (
	"strings"
	"time"
)

// DiskInfo provides basic information about the disk
type diskInfo struct {
	TrashSize     int           `json:"trash_size,omitempty"` // this and below are in bytes
	TotalSpace    int           `json:"total_space,omitempty"`
	UsedSpace     int           `json:"used_space,omitempty"`
	SystemFolders systemFolders `json:"system_folders,omitempty"`
	User          user          `json:"user,omitempty"`
	Revision      int64         `json:"revision,omitempty"`
}

// SystemFolders is a list of system folders on the disk
type systemFolders map[string]string

// Link contains URL to request Resource metadata
type link struct {
	Href      string `json:"href,omitempty"`
	Method    string `json:"method,omitempty"`
	Templated bool   `json:"templated,omitempty"`
}

// Resource holds information about the resource (either directory or file)
type resource struct {
	PubLicKey        string            `json:"public_key,omitempty"`
	PublicURL        string            `json:"public_url,omitempty"`
	Embedded         resourceList      `json:"_embedded,omitempty"`
	Name             string            `json:"name,omitempty"`
	Exif             map[string]string `json:"exif,omitempty"`            // seems to only appear in photos
	PhotosliceTime   time.Time         `json:"photoslice_time,omitempty"` // seems to only appear in photos
	DownloadLink     string            `json:"file,omitempty"`            // download link (for photos only?)
	PreviewLink      string            `json:"preview,omitempty"`         // preview link (for photos only?)
	MediaType        string            `json:"media_type,omitempty"`      // appears in photos and videos (only?)
	ResourceID       string            `json:"resource_id,omitempty"`     // unique id of a file
	Created          time.Time         `json:"created,omitempty"`
	Modified         time.Time         `json:"modified,omitempty"`
	CustomProperties map[string]string `json:"custom_properties,omitempty"`
	OriginPath       string            `json:"origin_path,omitempty"`
	Path             string            `json:"path,omitempty"`
	MD5              string            `json:"md5,omitempty"`
	SHA256           string            `json:"sha26,omitempty"`
	CommentIDs       commentIDs        `json:"comment_ids,omitempty"`      // undocumented :)
	Type             string            `json:"type,omitempty"`             // "dir" or "file"
	MimeType         string            `json:"mime_type,omitempty"`        // "image/jpeg", "video/mp4" etc.
	Size             int64             `json:"size,omitempty"`             // size in bytes (?)
	Revision         int64             `json:"revision,omitempty"`         // dunno?
	AntivirusStatus  string            `json:"antivirus_status,omitempty"` // "clean", "not-scanned" etc.

}

// ResourceList represents a list of resources
type resourceList struct {
	Sort      string     `json:"sort,omitempty"` // list is sorted by this field
	PubLicKey string     `json:"public_key,omitempty"`
	Items     []resource `json:"items,omitempty"`
	Path      string     `json:"path,omitempty"`
	Limit     int        `json:"limit,omitempty"`  // this max elements are in Items above
	Offset    int        `json:"offset,omitempty"` // offset from first resource in directory
	Total     int        `json:"total,omitempty"`  // total number of elements in directory
}

type commentIDs struct {
	PrivateResourceID string `json:"private_resource,omitempty"`
	PublicResourseID  string `json:"public_resource,omitempty"`
}

// FileResourceList is a flat list of all files on disk sorted alphabetically
type filesResourceList struct {
	Items  []resource `json:"items,omitempty"`
	Limit  int        `json:"limit,omitempty"`  // this max elements are in Items above
	Offset int        `json:"offset,omitempty"` // offset from first resource in directory
}

// LastUploadedResourceList is a list of uploaded files sorted by
// upload time from oldest to newest
type lastUploadedResourceList struct {
	Items []resource `json:"items,omitempty"`
	Limit int        `json:"limit,omitempty"`
}

// PublicResourcesList represents a list of publicly available resources
type publicResourcesList struct {
	Items  []resource `json:"items,omitempty"`
	Type   string     `json:"type,omitempty"`
	Limit  int        `json:"limit,omitempty"`
	Offset int        `json:"offset,omitempty"`
}

type operation struct {
	Status string `json:"status,omitempty"`
}

type user struct {
	Country string `json:"country,omitempty`
	Login   string `json:"login,omitempty`
	Name    string `json:"display_name,omitempty`
	UID     string `json:"uid,omitempty`
}

type errAPI struct {
	Message     string `json:"message,omitempty"`
	Description string `json:"description,omitempty"`
	Err         string `json:"error,omitempty"`
}

func (e *errAPI) Error() string {
	return strings.Join([]string{e.Message, e.Description, e.Err}, " ")
}
