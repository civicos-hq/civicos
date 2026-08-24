package issues

import (
	"errors"
	"testing"

	"github.com/civicos/community-service/internal/domain"
)

func baseInput() CreateInput {
	return CreateInput{
		Title:       "Streetlight outage",
		Description: "The whole street has been dark for a week.",
		Category:    domain.CategoryInfrastructure,
		CommunityID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
}

func TestCreateWithVideo(t *testing.T) {
	svc := NewService(&fakeIssueRepo{})
	in := baseInput()
	in.VideoURLs = []string{"clip.mp4"}
	in.VideoPosterURLs = []string{"poster.jpg"}
	in.VideoSizeBytes = []int64{4_200_000}

	issue, err := svc.Create(in, "reporter-id")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(issue.VideoURLs) != 1 || issue.VideoURLs[0] != "clip.mp4" {
		t.Errorf("VideoURLs = %v", issue.VideoURLs)
	}
	if len(issue.VideoPosterURLs) != 1 || issue.VideoPosterURLs[0] != "poster.jpg" {
		t.Errorf("VideoPosterURLs = %v", issue.VideoPosterURLs)
	}
	if len(issue.VideoSizeBytes) != 1 || issue.VideoSizeBytes[0] != 4_200_000 {
		t.Errorf("VideoSizeBytes = %v", issue.VideoSizeBytes)
	}
}

// A text-only report is first-class: no video fields, no error, and empty
// (not nil) slices so the JSON is [] rather than null.
func TestCreateWithoutVideoStaysFirstClass(t *testing.T) {
	svc := NewService(&fakeIssueRepo{})

	issue, err := svc.Create(baseInput(), "reporter-id")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if issue.VideoURLs == nil || len(issue.VideoURLs) != 0 {
		t.Errorf("VideoURLs = %v, want empty non-nil", issue.VideoURLs)
	}
	if issue.ImageURLs == nil || len(issue.ImageURLs) != 0 {
		t.Errorf("ImageURLs = %v, want empty non-nil", issue.ImageURLs)
	}
}

func TestCreateRejectsMultipleVideos(t *testing.T) {
	svc := NewService(&fakeIssueRepo{})
	in := baseInput()
	in.VideoURLs = []string{"a.mp4", "b.mp4"}

	_, err := svc.Create(in, "reporter-id")
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *AppError", err)
	}
	if appErr.Code != "TOO_MANY_VIDEOS" {
		t.Errorf("code = %q, want TOO_MANY_VIDEOS", appErr.Code)
	}
}

func TestCreateRejectsEmptyVideoReference(t *testing.T) {
	svc := NewService(&fakeIssueRepo{})
	in := baseInput()
	in.VideoURLs = []string{""}

	_, err := svc.Create(in, "reporter-id")
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *AppError", err)
	}
	if appErr.Code != "INVALID_VIDEO" {
		t.Errorf("code = %q, want INVALID_VIDEO", appErr.Code)
	}
}

// The three arrays are positional. A client that sends a video with no poster,
// or sends more posters than videos, must not produce arrays that disagree —
// readers index them together.
func TestVideoSidecarArraysAlwaysAlign(t *testing.T) {
	cases := []struct {
		name string
		in   CreateInput
	}{
		{"no poster or size", CreateInput{VideoURLs: []string{"clip.mp4"}}},
		{"poster only", CreateInput{
			VideoURLs:       []string{"clip.mp4"},
			VideoPosterURLs: []string{"poster.jpg"},
		}},
		{"more sidecars than videos", CreateInput{
			VideoURLs:       []string{"clip.mp4"},
			VideoPosterURLs: []string{"poster.jpg", "stray.jpg"},
			VideoSizeBytes:  []int64{100, 200, 300},
		}},
		{"sidecars with no video", CreateInput{
			VideoPosterURLs: []string{"orphan.jpg"},
			VideoSizeBytes:  []int64{999},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			videos, posters, sizes, err := normalizeVideos(tc.in)
			if err != nil {
				t.Fatalf("normalizeVideos: %v", err)
			}
			if len(posters) != len(videos) || len(sizes) != len(videos) {
				t.Fatalf("lengths disagree: videos=%d posters=%d sizes=%d",
					len(videos), len(posters), len(sizes))
			}
		})
	}
}
