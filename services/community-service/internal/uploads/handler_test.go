package uploads

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) (*gin.Engine, *LocalStorage, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	noop := func(c *gin.Context) { c.Next() }
	r := gin.New()
	v1 := r.Group("/v1")
	NewHandler(store).RegisterRoutes(v1.Group("/uploads"), noop, noop)
	NewVideoHandler(store).RegisterRoutes(v1.Group("/uploads"), noop, noop)
	return r, store, dir
}

// multipartBody builds a file part with an explicit Content-Type, which is what
// the validators read — Go's CreateFormFile hardcodes application/octet-stream.
func multipartBody(t *testing.T, filename, contentType string, content []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		`form-data; name="file"; filename="`+filename+`"`)
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}

func postFile(t *testing.T, r *gin.Engine, path, filename, contentType string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := multipartBody(t, filename, contentType, content)
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestVideoUpload(t *testing.T) {
	r, _, dir := newTestRouter(t)

	rec := postFile(t, r, "/v1/uploads/video", "pothole.mp4", "video/mp4", []byte("fake mp4 bytes"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"sizeBytes":14`) {
		t.Errorf("response missing sizeBytes: %s", body)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("stored %d files, want 1", len(entries))
	}
	// The stored name must be ours (UUID + extension), never the caller's.
	name := entries[0].Name()
	if strings.Contains(name, "pothole") {
		t.Errorf("stored name %q leaks the client filename", name)
	}
	if filepath.Ext(name) != ".mp4" {
		t.Errorf("stored name %q lost its extension", name)
	}
}

func TestVideoUploadRejections(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		size        int
		wantStatus  int
		wantCode    string
	}{
		{"oversized", "big.mp4", "video/mp4", maxVideoBytes + 1, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE"},
		{"bad extension", "clip.avi", "video/mp4", 16, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE"},
		{"bad content type", "clip.mp4", "application/octet-stream", 16, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE"},
		{"image through the video endpoint", "photo.jpg", "image/jpeg", 16, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, _, dir := newTestRouter(t)
			rec := postFile(t, r, "/v1/uploads/video", tc.filename, tc.contentType, bytes.Repeat([]byte("a"), tc.size))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), `"code":"`+tc.wantCode+`"`) {
				t.Errorf("body = %s, want code %s", rec.Body.String(), tc.wantCode)
			}
			// A rejected upload must not leave a file behind.
			entries, _ := os.ReadDir(dir)
			if len(entries) != 0 {
				t.Errorf("rejected upload wrote %d files", len(entries))
			}
		})
	}
}

func TestVideoUploadRequiresFile(t *testing.T) {
	r, _, _ := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/uploads/video", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "FILE_REQUIRED") {
		t.Errorf("body = %s, want FILE_REQUIRED", rec.Body.String())
	}
}

// The video endpoint must not become a second way in for images, and the image
// endpoint must keep rejecting videos.
func TestEndpointsDoNotOverlap(t *testing.T) {
	r, _, _ := newTestRouter(t)

	rec := postFile(t, r, "/v1/uploads", "clip.mp4", "video/mp4", []byte("x"))
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("image endpoint accepted a video: %d %s", rec.Code, rec.Body.String())
	}

	rec = postFile(t, r, "/v1/uploads", "photo.png", "image/png", []byte("x"))
	if rec.Code != http.StatusCreated {
		t.Errorf("image endpoint broke: %d %s", rec.Code, rec.Body.String())
	}
}

func TestServeRoundTripAndContentType(t *testing.T) {
	r, _, _ := newTestRouter(t)

	content := []byte("fake webm bytes")
	rec := postFile(t, r, "/v1/uploads/video", "clip.webm", "video/webm", content)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload failed: %d %s", rec.Code, rec.Body.String())
	}
	name := storedName(t, rec.Body.String())

	req := httptest.NewRequest(http.MethodGet, "/v1/uploads/"+name, nil)
	got := httptest.NewRecorder()
	r.ServeHTTP(got, req)

	if got.Code != http.StatusOK {
		t.Fatalf("serve status = %d, want 200", got.Code)
	}
	if !bytes.Equal(got.Body.Bytes(), content) {
		t.Errorf("served bytes differ from uploaded bytes")
	}
	// Go's MIME table doesn't know .webm and alpine has no /etc/mime.types,
	// so this header has to come from contentTypeFor or playback breaks.
	if ct := got.Header().Get("Content-Type"); ct != "video/webm" {
		t.Errorf("Content-Type = %q, want video/webm", ct)
	}
}

// Range support is what lets a viewer scrub a video without downloading it all.
func TestServeSupportsRangeRequests(t *testing.T) {
	r, _, _ := newTestRouter(t)

	content := []byte("0123456789abcdef")
	rec := postFile(t, r, "/v1/uploads/video", "clip.mp4", "video/mp4", content)
	name := storedName(t, rec.Body.String())

	req := httptest.NewRequest(http.MethodGet, "/v1/uploads/"+name, nil)
	req.Header.Set("Range", "bytes=4-7")
	got := httptest.NewRecorder()
	r.ServeHTTP(got, req)

	if got.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", got.Code)
	}
	if got.Body.String() != "4567" {
		t.Errorf("body = %q, want %q", got.Body.String(), "4567")
	}
}

func TestServeRejectsTraversal(t *testing.T) {
	r, _, dir := newTestRouter(t)

	// A file the traversal would reach if the guard failed.
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	// Gin's :filename param never matches across "/", so a raw "../secret.txt"
	// is a routing miss rather than a read. The guard exists for the encoded
	// and single-segment forms that DO reach the handler.
	for _, name := range []string{
		"..",
		"%2e%2e%2fsecret.txt",
		"..%2Fsecret.txt",
		".hidden",
		"a..b.mp4",
	} {
		req := httptest.NewRequest(http.MethodGet, "/v1/uploads/"+name, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Errorf("serve(%q) returned 200 — traversal guard failed", name)
		}
		if strings.Contains(rec.Body.String(), "top secret") {
			t.Fatalf("serve(%q) leaked file contents", name)
		}
	}
}

func TestServeMissingFileIs404(t *testing.T) {
	r, _, _ := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/uploads/6ba7b810-9dad-11d1-80b4-00c04fd430c8.mp4", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// LocalStorage guards paths itself, independently of the handlers.
func TestLocalStorageRefusesUnsafeNames(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	for _, name := range []string{"../escape.mp4", "sub/dir.mp4", "..", ""} {
		if err := store.Save(name, strings.NewReader("x")); !errors.Is(err, ErrNotFound) {
			t.Errorf("Save(%q) err = %v, want ErrNotFound", name, err)
		}
		if _, err := store.Open(name); !errors.Is(err, ErrNotFound) {
			t.Errorf("Open(%q) err = %v, want ErrNotFound", name, err)
		}
	}

	// Nothing escaped the directory.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("unsafe saves wrote %d files", len(entries))
	}
}

func TestLocalStorageOpenMissing(t *testing.T) {
	store, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	if _, err := store.Open("nope.mp4"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Open(missing) err = %v, want ErrNotFound", err)
	}
}

// storedName pulls the generated filename out of an upload response body.
func storedName(t *testing.T, body string) string {
	t.Helper()
	const key = `"filename":"`
	i := strings.Index(body, key)
	if i < 0 {
		t.Fatalf("no filename in response: %s", body)
	}
	rest := body[i+len(key):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		t.Fatalf("malformed filename in response: %s", body)
	}
	return rest[:j]
}
