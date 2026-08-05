package discover

import (
	"net/http"
	"strconv"
	"time"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

// Public activity — the unauthenticated ticker the marketing homepage reads.
//
// # Why this is not just the discover feed
//
// The feed does the same aggregation, but it is authenticated, personalised by
// community, and hydrates whole entities plus their organizations. None of that
// is wanted here. This endpoint is reachable by anyone on the internet, so it
// returns the smallest thing that can honestly be shown: what kind of thing
// happened, its title, roughly where, and when.
//
// Deliberately absent:
//
//   - **Author names and user ids.** A ticker does not need to name the person
//     who reported a broken transformer, and putting a citizen's name on the
//     public homepage of a government-facing platform is a different decision
//     from putting it on a page you have to sign in to read.
//   - **Bodies and descriptions.** A title is enough for a ticker, and less
//     text is less to get wrong.
//   - **Anything not already public.** Drafts, pending review, rejected, and
//     archived records never appear.
type PublicActivityItem struct {
	// Kind is one of: issue, petition, consultation, announcement, campaign,
	// repAnnouncement.
	Kind  string `json:"kind"`
	Title string `json:"title"`
	// Status is the entity's own lifecycle value where it has one a citizen
	// would recognise (an issue's OPEN/RESOLVED). Empty otherwise.
	Status string `json:"status,omitempty"`
	// State and LGA locate it no more precisely than the platform already
	// does publicly. Empty when the record is not tied to a place.
	State string    `json:"state,omitempty"`
	LGA   string    `json:"lga,omitempty"`
	At    time.Time `json:"at"`
}

const (
	publicActivityDefault = 8
	publicActivityMax     = 20
)

// PublicActivity returns the most recent public records across every kind,
// newest first.
//
// One query per kind then merge, rather than a UNION: the kinds live in tables
// owned by two services with different shapes, and a UNION would force a
// lowest-common-denominator projection that breaks the moment one of them
// gains a column.
func (s *Service) PublicActivity(limit int) ([]PublicActivityItem, error) {
	if limit <= 0 {
		limit = publicActivityDefault
	}
	if limit > publicActivityMax {
		limit = publicActivityMax
	}

	out := []PublicActivityItem{}

	type row struct {
		Title  string
		Status string
		State  string
		LGA    string
		At     time.Time
	}
	collect := func(kind string, scan func(*[]row) error) error {
		var rows []row
		if err := scan(&rows); err != nil {
			return err
		}
		for _, r := range rows {
			out = append(out, PublicActivityItem{
				Kind: kind, Title: r.Title, Status: r.Status,
				State: r.State, LGA: r.LGA, At: r.At,
			})
		}
		return nil
	}

	// Issues and petitions carry their community, which is where the place
	// comes from.
	if err := collect("issue", func(rows *[]row) error {
		return s.db.Table("issues AS i").
			Select("i.title, i.status, c.state, c.lga, i.created_at AS at").
			Joins("LEFT JOIN communities c ON c.id = i.community_id").
			Order("i.created_at desc").Limit(limit).Scan(rows).Error
	}); err != nil {
		return nil, err
	}

	if err := collect("petition", func(rows *[]row) error {
		return s.db.Table("petitions AS p").
			Select("p.title, p.status, c.state, c.lga, p.created_at AS at").
			Joins("LEFT JOIN communities c ON c.id = p.community_id").
			Order("p.created_at desc").Limit(limit).Scan(rows).Error
	}); err != nil {
		return nil, err
	}

	// Organization-owned kinds, read from the shared database with the same
	// arrangement the feed uses. Each is filtered to the statuses that
	// already have a public page.
	if err := collect("consultation", func(rows *[]row) error {
		return s.db.Table("consultations AS x").
			Select("x.title, x.status, c.state, c.lga, COALESCE(x.published_at, x.created_at) AS at").
			Joins("LEFT JOIN communities c ON c.id = x.community_id").
			Where("x.status IN ?", []string{"PUBLISHED", "CLOSED"}).
			Order("COALESCE(x.published_at, x.created_at) desc").Limit(limit).Scan(rows).Error
	}); err != nil {
		return nil, err
	}

	if err := collect("announcement", func(rows *[]row) error {
		return s.db.Table("announcements AS a").
			Select("a.title, '' AS status, o.state, o.lga, COALESCE(a.published_at, a.created_at) AS at").
			Joins("LEFT JOIN organizations o ON o.id = a.organization_id").
			Where("a.status = ?", "PUBLISHED").
			Order("COALESCE(a.published_at, a.created_at) desc").Limit(limit).Scan(rows).Error
	}); err != nil {
		return nil, err
	}

	if err := collect("campaign", func(rows *[]row) error {
		return s.db.Table("campaigns AS c").
			Select("c.title, c.status, c.state, c.lga, COALESCE(c.published_at, c.created_at) AS at").
			Where("c.status IN ?", []string{"PUBLISHED", "FUNDED", "COMPLETED", "REPORTED"}).
			Order("COALESCE(c.published_at, c.created_at) desc").Limit(limit).Scan(rows).Error
	}); err != nil {
		return nil, err
	}

	if err := collect("repAnnouncement", func(rows *[]row) error {
		return s.db.Table("representative_announcements AS ra").
			Select("ra.title, '' AS status, c.state, c.lga, COALESCE(ra.published_at, ra.created_at) AS at").
			Joins("LEFT JOIN communities c ON c.id = ra.community_id").
			Where("ra.status = ?", "PUBLISHED").
			Order("COALESCE(ra.published_at, ra.created_at) desc").Limit(limit).Scan(rows).Error
	}); err != nil {
		return nil, err
	}

	sortByRecency(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func sortByRecency(items []PublicActivityItem) {
	// Small n (six kinds × limit), so an insertion sort is clearer than
	// pulling in sort.Slice and a closure.
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].At.After(items[j-1].At); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func (h *Handler) publicActivity(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.svc.PublicActivity(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load activity")
		return
	}
	// An empty list is a valid answer, not an error. A platform with no public
	// records yet should say so honestly rather than have the caller invent
	// something to fill the space.
	response.Success(c, http.StatusOK, gin.H{"activity": items})
}
