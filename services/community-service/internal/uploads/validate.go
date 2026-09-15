package uploads

import (
	"net/http"
	"path/filepath"
	"strings"
)

const (
	maxImageBytes = 5 * 1024 * 1024
	maxVideoBytes = 10 * 1024 * 1024
)

var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// videoExts and videoTypes are checked together: a file must have BOTH an
// allowed extension AND an allowed Content-Type. Either alone is trivially
// forged, and the pair is what the image handler has always required.
var videoExts = map[string]bool{
	".mp4":  true,
	".webm": true,
	".mov":  true,
}

var videoTypes = map[string]bool{
	"video/mp4":       true,
	"video/webm":      true,
	"video/quicktime": true,
}

// contentTypeFor maps a stored extension to the Content-Type we serve it with.
// Go's built-in MIME table doesn't know .mp4/.webm/.mov, and the alpine images
// the services ship in have no /etc/mime.types to fall back on — without this,
// http.ServeContent would sniff and could hand the browser something it won't
// play.
var contentTypeFor = map[string]string{
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mov":  "video/quicktime",
}

// uploadError is a validation failure carrying the HTTP status and the stable
// error code the client switches on.
type uploadError struct {
	Status  int
	Code    string
	Message string
}

// validateImage enforces the image rules: 5MB, known raster extension, and an
// image/* Content-Type.
func validateImage(size int64, filename, contentType string) *uploadError {
	if size > maxImageBytes {
		return &uploadError{http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Files must be 5MB or smaller"}
	}
	if !imageExts[extOf(filename)] {
		return &uploadError{http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only JPG, PNG, GIF, and WEBP images are allowed"}
	}
	if !strings.HasPrefix(contentType, "image/") {
		return &uploadError{http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only image files are allowed"}
	}
	return nil
}

// validateVideo enforces the video rules: 10MB, one of three container
// extensions, and a matching Content-Type from a closed allowlist.
//
// DURATION IS NOT CHECKED HERE. The 20-second cap is enforced in the browser
// by reading HTMLVideoElement.duration before upload. Reading duration
// server-side would mean shelling out to ffprobe or parsing container
// metadata, and transcoding is explicitly out of scope for this change. The
// split is therefore: the SERVER guarantees size and type, the CLIENT
// guarantees duration. A crafted request can get a 45-second 8MB clip past
// us. That is an accepted gap — the size cap already bounds the real cost
// (storage and the viewer's mobile data), and duration is a UX rule rather
// than a safety one. Revisit if and when server-side media processing arrives.
func validateVideo(size int64, filename, contentType string) *uploadError {
	if size > maxVideoBytes {
		return &uploadError{http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Videos must be 10MB or smaller"}
	}
	if !videoExts[extOf(filename)] {
		return &uploadError{http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only MP4, WEBM, and MOV videos are allowed"}
	}
	if !videoTypes[normalizeContentType(contentType)] {
		return &uploadError{http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Only MP4, WEBM, and MOV videos are allowed"}
	}
	return nil
}

func extOf(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// normalizeContentType strips any parameters ("video/mp4; codecs=avc1") and
// lowercases, so the allowlist compares against the bare media type.
func normalizeContentType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

// safeStoredName reports whether name is a plain stored filename — no path
// separators, no traversal, no leading dot. Anything reaching the filesystem
// goes through here first.
func safeStoredName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	// Rejects "." and dotfiles; every name we generate is a UUID.
	if strings.HasPrefix(name, ".") {
		return false
	}
	// Null bytes truncate paths in some syscalls; never let one through.
	return !strings.ContainsRune(name, 0)
}
