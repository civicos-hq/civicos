package uploads

import "testing"

func TestValidateVideo(t *testing.T) {
	tests := []struct {
		name        string
		size        int64
		filename    string
		contentType string
		wantCode    string
	}{
		{"mp4 accepted", 1 << 20, "pothole.mp4", "video/mp4", ""},
		{"webm accepted", 1 << 20, "burst-pipe.webm", "video/webm", ""},
		{"mov accepted with quicktime type", 1 << 20, "flood.mov", "video/quicktime", ""},
		{"extension case is ignored", 1 << 20, "FLOOD.MP4", "video/mp4", ""},
		{"content-type parameters are ignored", 1 << 20, "clip.mp4", "video/mp4; codecs=avc1.42E01E", ""},
		{"exactly at the cap is allowed", maxVideoBytes, "clip.mp4", "video/mp4", ""},

		{"one byte over the cap is rejected", maxVideoBytes + 1, "clip.mp4", "video/mp4", "FILE_TOO_LARGE"},
		{"size is checked before type", maxVideoBytes + 1, "clip.exe", "application/octet-stream", "FILE_TOO_LARGE"},

		{"disallowed extension", 1 << 20, "clip.avi", "video/mp4", "INVALID_FILE_TYPE"},
		{"no extension", 1 << 20, "clip", "video/mp4", "INVALID_FILE_TYPE"},
		{"image extension", 1 << 20, "photo.jpg", "video/mp4", "INVALID_FILE_TYPE"},

		// The pair matters: either half alone is trivially forged, so a good
		// extension with a bad type (and vice versa) must fail.
		{"good extension, disallowed type", 1 << 20, "clip.mp4", "application/octet-stream", "INVALID_FILE_TYPE"},
		{"good extension, empty type", 1 << 20, "clip.mp4", "", "INVALID_FILE_TYPE"},
		{"good extension, image type", 1 << 20, "clip.mp4", "image/png", "INVALID_FILE_TYPE"},
		{"video/* prefix is not enough", 1 << 20, "clip.mp4", "video/x-msvideo", "INVALID_FILE_TYPE"},
		{"double extension resolves to the last one", 1 << 20, "clip.mp4.exe", "video/mp4", "INVALID_FILE_TYPE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVideo(tc.size, tc.filename, tc.contentType)
			gotCode := ""
			if err != nil {
				gotCode = err.Code
			}
			if gotCode != tc.wantCode {
				t.Fatalf("validateVideo(%d, %q, %q) code = %q, want %q",
					tc.size, tc.filename, tc.contentType, gotCode, tc.wantCode)
			}
		})
	}
}

// The image path predates videos and must keep behaving exactly as it did.
func TestValidateImage(t *testing.T) {
	tests := []struct {
		name        string
		size        int64
		filename    string
		contentType string
		wantCode    string
	}{
		{"jpg accepted", 1 << 20, "photo.jpg", "image/jpeg", ""},
		{"webp accepted", 1 << 20, "photo.webp", "image/webp", ""},
		{"exactly at the cap is allowed", maxImageBytes, "photo.png", "image/png", ""},
		{"over the cap", maxImageBytes + 1, "photo.png", "image/png", "FILE_TOO_LARGE"},
		{"disallowed extension", 1 << 20, "photo.svg", "image/svg+xml", "INVALID_FILE_TYPE"},
		{"non-image content type", 1 << 20, "photo.png", "text/html", "INVALID_FILE_TYPE"},
		{"a video is not an image", 1 << 20, "clip.mp4", "video/mp4", "INVALID_FILE_TYPE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateImage(tc.size, tc.filename, tc.contentType)
			gotCode := ""
			if err != nil {
				gotCode = err.Code
			}
			if gotCode != tc.wantCode {
				t.Fatalf("validateImage(%d, %q, %q) code = %q, want %q",
					tc.size, tc.filename, tc.contentType, gotCode, tc.wantCode)
			}
		})
	}
}

// The video cap must actually be larger than the image cap, and both must stay
// bounded — a fat-fingered constant here is a silent capacity change.
func TestUploadCaps(t *testing.T) {
	if maxImageBytes != 5*1024*1024 {
		t.Errorf("image cap = %d, want 5MB", maxImageBytes)
	}
	if maxVideoBytes != 10*1024*1024 {
		t.Errorf("video cap = %d, want 10MB", maxVideoBytes)
	}
}

func TestSafeStoredName(t *testing.T) {
	safe := []string{
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8.mp4",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8.jpg",
		"plain.webm",
	}
	for _, name := range safe {
		if !safeStoredName(name) {
			t.Errorf("safeStoredName(%q) = false, want true", name)
		}
	}

	unsafe := []string{
		"",
		"..",
		"../../etc/passwd",
		"..%2f..%2fetc%2fpasswd/x", // contains a separator
		"sub/dir.mp4",
		`sub\dir.mp4`,
		"a..b.mp4",     // ".." anywhere is refused, not just as a segment
		".env",         // dotfile
		".",            //
		"clip.mp4\x00", // null byte truncation
	}
	for _, name := range unsafe {
		if safeStoredName(name) {
			t.Errorf("safeStoredName(%q) = true, want false", name)
		}
	}

	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	if safeStoredName(string(long)) {
		t.Error("safeStoredName(256 chars) = true, want false")
	}
}
