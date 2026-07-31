package progress

import (
	"errors"
	"testing"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type fakeStore struct{ created []*domain.ProgressUpdate }

func (f *fakeStore) Find(ListFilters) ([]domain.ProgressUpdate, error) { return nil, nil }
func (f *fakeStore) FindByID(string) (*domain.ProgressUpdate, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) Create(p *domain.ProgressUpdate) error {
	f.created = append(f.created, p)
	return nil
}
func (f *fakeStore) Delete(string) error { return nil }

type fakeCampaigns struct{ byID map[string]*domain.Campaign }

func (f *fakeCampaigns) Get(id string) (*domain.Campaign, error) {
	if c, ok := f.byID[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func isCode(err error, code string) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == code
}

func ptr(s string) *string { return &s }

func newSvc() (*fakeStore, *Service) {
	st := &fakeStore{}
	camps := &fakeCampaigns{byID: map[string]*domain.Campaign{
		"camp-mine":  {ID: "camp-mine", OrganizationID: "org-1", Title: "Flood relief", Slug: "flood-relief"},
		"camp-other": {ID: "camp-other", OrganizationID: "org-2", Title: "Someone else's", Slug: "other"},
	}}
	return st, NewService(st, nil).WithCampaigns(camps)
}

// THE authorisation property for this feature.
//
// The handler checks the caller administers the org in the URL, but
// campaignId arrives in the request BODY. Without an ownership check, an
// admin of one organization could publish updates onto another
// organization's campaign page, under its name, to its donors.
func TestCreate_RefusesACampaignBelongingToAnotherOrg(t *testing.T) {
	st, svc := newSvc()

	_, err := svc.Create("org-1", CreateInput{
		CampaignID: ptr("camp-other"), Body: "We spent the money well.",
	}, "user-1", "Ada")

	if err == nil {
		t.Fatal("posted an update onto another organization's campaign")
	}
	// NOT_FOUND, not FORBIDDEN: a 403 confirms the campaign exists and lets
	// someone probe for ids.
	if !isCode(err, "CAMPAIGN_NOT_FOUND") {
		t.Fatalf("want CAMPAIGN_NOT_FOUND, got %v", err)
	}
	if len(st.created) != 0 {
		t.Fatal("a record was written despite the rejection")
	}
}

func TestCreate_AcceptsTheOrgsOwnCampaign(t *testing.T) {
	st, svc := newSvc()

	p, err := svc.Create("org-1", CreateInput{
		CampaignID: ptr("camp-mine"), Title: ptr("Boreholes complete"),
		Body:        "Two of three boreholes are delivering water.",
		Attachments: []string{"https://cdn.example.org/borehole.jpg"},
	}, "user-1", "Ada")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.CampaignID == nil || *p.CampaignID != "camp-mine" {
		t.Fatalf("campaign not attached: %+v", p.CampaignID)
	}
	if len(p.AttachmentURLs) != 1 {
		t.Fatalf("attachments lost: %v", p.AttachmentURLs)
	}
	if len(st.created) != 1 {
		t.Fatal("not persisted")
	}
}

// A missing campaign must not be silently accepted as an unattached update.
func TestCreate_UnknownCampaignIsRejected(t *testing.T) {
	_, svc := newSvc()
	if _, err := svc.Create("org-1", CreateInput{
		CampaignID: ptr("does-not-exist"), Body: "x",
	}, "u", "n"); !isCode(err, "CAMPAIGN_NOT_FOUND") {
		t.Fatalf("want CAMPAIGN_NOT_FOUND, got %v", err)
	}
}

// Without campaign support wired, campaignId must be refused rather than
// accepted unchecked — an unverifiable target is worse than an unsupported
// one.
func TestCreate_RefusesCampaignTargetWhenLookupIsUnavailable(t *testing.T) {
	st := &fakeStore{}
	svc := NewService(st, nil) // no WithCampaigns

	if _, err := svc.Create("org-1", CreateInput{
		CampaignID: ptr("camp-mine"), Body: "x",
	}, "u", "n"); !isCode(err, "INVALID_TARGET") {
		t.Fatalf("want INVALID_TARGET, got %v", err)
	}
	if len(st.created) != 0 {
		t.Fatal("wrote an unverified campaign update")
	}
}

// An update belonging to two things at once would appear on both feeds
// saying different things to different audiences.
func TestCreate_ExactlyOneTarget(t *testing.T) {
	_, svc := newSvc()

	cases := []CreateInput{
		{Body: "x"}, // none
		{Body: "x", IssueID: ptr("i-1"), ProjectID: ptr("p-1")},
		{Body: "x", IssueID: ptr("i-1"), CampaignID: ptr("camp-mine")},
		{Body: "x", ProjectID: ptr("p-1"), CampaignID: ptr("camp-mine")},
		{Body: "x", IssueID: ptr("i-1"), ProjectID: ptr("p-1"), CampaignID: ptr("camp-mine")},
	}
	for i, in := range cases {
		if _, err := svc.Create("org-1", in, "u", "n"); !isCode(err, "INVALID_TARGET") {
			t.Errorf("case %d: want INVALID_TARGET, got %v", i, err)
		}
	}
}

// Existing issue and project updates must keep working untouched.
func TestCreate_IssueAndProjectUpdatesStillWork(t *testing.T) {
	_, svc := newSvc()

	if _, err := svc.Create("org-1", CreateInput{IssueID: ptr("i-1"), Body: "x"}, "u", "n"); err != nil {
		t.Fatalf("issue update broke: %v", err)
	}
	if _, err := svc.Create("org-1", CreateInput{ProjectID: ptr("p-1"), Body: "x"}, "u", "n"); err != nil {
		t.Fatalf("project update broke: %v", err)
	}
}

// Clients map over attachments without a guard.
func TestCreate_AttachmentsAreNeverNil(t *testing.T) {
	_, svc := newSvc()
	p, err := svc.Create("org-1", CreateInput{IssueID: ptr("i-1"), Body: "x"}, "u", "n")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.AttachmentURLs == nil {
		t.Fatal("attachmentUrls is nil; it must serialise as []")
	}
}
