package communities

import (
	"strings"
	"testing"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type fakeCommunityRepo struct {
	items []domain.Community
}

// Search mirrors the real repository closely enough to exercise the
// service's paging and defaults: same filters, same prefix-first ordering,
// total counted before the page is cut.
func (f *fakeCommunityRepo) Search(p SearchParams) ([]domain.Community, int64, error) {
	matched := make([]domain.Community, 0, len(f.items))
	term := strings.ToLower(strings.TrimSpace(p.Query))
	wantIDs := make(map[string]struct{}, len(p.IDs))
	for _, id := range p.IDs {
		wantIDs[id] = struct{}{}
	}
	for _, item := range f.items {
		if len(wantIDs) > 0 {
			if _, ok := wantIDs[item.ID]; !ok {
				continue
			}
		}
		if p.State != "" && item.State != p.State {
			continue
		}
		if p.LGA != "" && item.LGA != p.LGA {
			continue
		}
		if term != "" {
			hay := strings.ToLower(item.Name + " " + item.LGA + " " + item.State)
			if item.Description != nil {
				hay += " " + strings.ToLower(*item.Description)
			}
			if !strings.Contains(hay, term) {
				continue
			}
		}
		matched = append(matched, item)
	}

	total := int64(len(matched))

	if term != "" {
		prefixed := make([]domain.Community, 0, len(matched))
		rest := make([]domain.Community, 0, len(matched))
		for _, item := range matched {
			if strings.HasPrefix(strings.ToLower(item.Name), term) {
				prefixed = append(prefixed, item)
			} else {
				rest = append(rest, item)
			}
		}
		matched = append(prefixed, rest...)
	}

	if p.Offset >= len(matched) {
		return []domain.Community{}, total, nil
	}
	matched = matched[p.Offset:]
	if p.Limit > 0 && p.Limit < len(matched) {
		matched = matched[:p.Limit]
	}
	return matched, total, nil
}

func (f *fakeCommunityRepo) FindByID(id string) (*domain.Community, error) {
	for _, item := range f.items {
		if item.ID == id {
			found := item
			return &found, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeCommunityRepo) FindByIDs(ids []string) ([]domain.Community, error) {
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make([]domain.Community, 0, len(ids))
	for _, item := range f.items {
		if _, ok := want[item.ID]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeCommunityRepo) Create(c *domain.Community) error {
	f.items = append(f.items, *c)
	return nil
}

func seed(t *testing.T, svc *Service, name, slug, state, lga string) *domain.Community {
	t.Helper()
	created, err := svc.Create(CreateInput{Name: name, Slug: slug, State: state, LGA: lga}, "user-1")
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return created
}

func TestCreateAndListCommunities(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewService(repo)
	desc := "A vibrant local community"

	created, err := svc.Create(CreateInput{
		Name:        "Lekki",
		Slug:        "lekki",
		State:       "Lagos",
		LGA:         "Eti-Osa",
		Description: &desc,
	}, "user-1")
	if err != nil {
		t.Fatalf("create community: %v", err)
	}
	if created.Slug != "lekki" {
		t.Fatalf("expected slug lekki, got %s", created.Slug)
	}

	all, err := svc.List(SearchParams{})
	if err != nil {
		t.Fatalf("list communities: %v", err)
	}
	if len(all.Communities) != 1 {
		t.Fatalf("expected 1 community, got %d", len(all.Communities))
	}
	if all.Total != 1 {
		t.Fatalf("expected total 1, got %d", all.Total)
	}

	// State/LGA filters must narrow the result.
	filtered, err := svc.List(SearchParams{State: "Lagos", LGA: "Eti-Osa"})
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(filtered.Communities) != 1 {
		t.Fatalf("expected 1 filtered community, got %d", len(filtered.Communities))
	}
	empty, err := svc.List(SearchParams{State: "Rivers"})
	if err != nil {
		t.Fatalf("list mismatched state: %v", err)
	}
	if len(empty.Communities) != 0 {
		t.Fatalf("expected 0 communities in Rivers, got %d", len(empty.Communities))
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get community: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected community id %s, got %s", created.ID, got.ID)
	}
}

// The scenario the search was added for: several universities in one city,
// spread across different LGAs, none of them findable by drill-down unless
// the student already knows which LGA their campus sits in.
func TestSearchFindsUniversitiesAcrossLGAs(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewService(repo)

	seed(t, svc, "University of Abuja", "university-of-abuja", "FCT", "Gwagwalada")
	seed(t, svc, "Nile University of Nigeria", "nile-university", "FCT", "Abuja Municipal")
	seed(t, svc, "Baze University", "baze-university", "FCT", "Abuja Municipal")
	seed(t, svc, "Gwagwalada", "gwagwalada", "FCT", "Gwagwalada")
	seed(t, svc, "Lekki", "lekki", "Lagos", "Eti-Osa")

	res, err := svc.List(SearchParams{Query: "university"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.Total != 3 {
		t.Fatalf("expected 3 universities, got %d", res.Total)
	}
	// All three must come back despite living in two different LGAs.
	names := map[string]bool{}
	for _, c := range res.Communities {
		names[c.Name] = true
	}
	for _, want := range []string{"University of Abuja", "Nile University of Nigeria", "Baze University"} {
		if !names[want] {
			t.Fatalf("expected %q in results, got %v", want, names)
		}
	}

	// A prefix match must outrank an infix one.
	res, err = svc.List(SearchParams{Query: "Baze"})
	if err != nil {
		t.Fatalf("search baze: %v", err)
	}
	if len(res.Communities) == 0 || res.Communities[0].Name != "Baze University" {
		t.Fatalf("expected Baze University first, got %+v", res.Communities)
	}

	// Searching a place name should find the communities in it, so the
	// wizard's fallback path still works through the same box.
	res, err = svc.List(SearchParams{Query: "Gwagwalada"})
	if err != nil {
		t.Fatalf("search gwagwalada: %v", err)
	}
	if res.Total != 2 {
		t.Fatalf("expected 2 Gwagwalada matches (the LGA community + UniAbuja), got %d", res.Total)
	}
}

func TestListPaginates(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewService(repo)
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
		seed(t, svc, name, strings.ToLower(name), "FCT", "Abuja Municipal")
	}

	first, err := svc.List(SearchParams{Limit: 2})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Communities) != 2 || first.Total != 4 {
		t.Fatalf("expected 2 of 4, got %d of %d", len(first.Communities), first.Total)
	}

	second, err := svc.List(SearchParams{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Communities) != 2 {
		t.Fatalf("expected 2 on second page, got %d", len(second.Communities))
	}
	if second.Communities[0].ID == first.Communities[0].ID {
		t.Fatal("second page repeated the first page")
	}

	// An unbounded request must still come back bounded.
	capped, err := svc.List(SearchParams{Limit: 5000})
	if err != nil {
		t.Fatalf("capped: %v", err)
	}
	if capped.Limit != MaxPageSize {
		t.Fatalf("expected limit capped to %d, got %d", MaxPageSize, capped.Limit)
	}

	defaulted, err := svc.List(SearchParams{})
	if err != nil {
		t.Fatalf("defaulted: %v", err)
	}
	if defaulted.Limit != DefaultPageSize {
		t.Fatalf("expected default limit %d, got %d", DefaultPageSize, defaulted.Limit)
	}
}

// The browse page renders "communities you joined" from membership IDs,
// which need not appear on the page of search results currently shown.
func TestListByIDsIgnoresPaging(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewService(repo)
	seed(t, svc, "Alpha", "alpha", "FCT", "Bwari")
	wanted := seed(t, svc, "Zulu", "zulu", "Lagos", "Ikeja")

	// "Zulu" sorts last and would fall off the first page of a plain list.
	res, err := svc.List(SearchParams{IDs: []string{wanted.ID}, Limit: 1})
	if err != nil {
		t.Fatalf("list by ids: %v", err)
	}
	if len(res.Communities) != 1 || res.Communities[0].ID != wanted.ID {
		t.Fatalf("expected only Zulu, got %+v", res.Communities)
	}
}

func TestResolveRejectsUnknownCommunity(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewService(repo)
	a := seed(t, svc, "University of Abuja", "university-of-abuja", "FCT", "Gwagwalada")

	if _, err := svc.Resolve([]string{a.ID}); err != nil {
		t.Fatalf("resolve known: %v", err)
	}

	// A batch join must validate completely or write nothing, so one bad ID
	// fails the whole set rather than silently joining the good ones.
	_, err := svc.Resolve([]string{a.ID, "00000000-0000-0000-0000-000000000000"})
	if err == nil {
		t.Fatal("expected unknown community to fail resolution")
	}
	var appErr *AppError
	if !errorsAs(err, &appErr) || appErr.Code != "COMMUNITY_NOT_FOUND" {
		t.Fatalf("expected COMMUNITY_NOT_FOUND, got %v", err)
	}
}

// errorsAs keeps the test readable without importing errors just for one call.
func errorsAs(err error, target **AppError) bool {
	if e, ok := err.(*AppError); ok {
		*target = e
		return true
	}
	return false
}
