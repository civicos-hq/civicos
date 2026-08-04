package campaignai

import (
	"fmt"
	"strings"
	"time"

	"github.com/civicos/civicai-service/internal/gemini"
)

// deliberateTimeout applies to every campaign surface except classification.
//
// These prompts carry a full campaign fact sheet and ask for a nested
// response schema, and they are all triggered by someone who clicked a button
// and is watching a spinner — an org admin drafting a report, a reviewer
// opening a queue item. The 15s default is tuned for a citizen mid-form,
// where a slow answer is worse than none; here a timeout is a 502 on the work
// they actually came to do. Observed: risk assessment exceeded 15s in normal
// operation, intermittently, which is the worst way for a limit to be wrong.
const deliberateTimeout = 45 * time.Second

// Categories MUST stay in sync with organization-service domain.CampaignCategory.
// Gemini is constrained to this exact set via the response schema, so a value
// added there and not here becomes unreachable rather than wrong.
var Categories = []string{
	"EMERGENCY_RELIEF",
	"COMMUNITY_DEVELOPMENT",
	"EDUCATION",
	"HEALTHCARE",
	"ENVIRONMENT",
	"AGRICULTURE",
	"OTHER",
}

// Provenance is embedded in every response. Principle 2 of the CivicAI plan:
// AI output is provenance-tagged, and the UI shows an "AI-generated · review
// before publishing" badge. A caller cannot render one of these without also
// having the model name and timestamp to hand.
type Provenance struct {
	Model       string    `json:"model"`
	GeneratedAt time.Time `json:"generatedAt"`
	// Advisory is always true. It exists so the field is visible in every
	// response body and in the API docs — a client that ever finds itself
	// wanting to act on one of these automatically has to read the word
	// first.
	Advisory bool `json:"advisory"`
}

func (s *Service) provenance() Provenance {
	return Provenance{Model: s.ai.Model(), GeneratedAt: time.Now().UTC(), Advisory: true}
}

// Service holds the six campaign surfaces. One struct rather than six because
// they share the same client and the same source reader; the tasks are kept
// apart in separate files.
type Service struct {
	ai  *gemini.Client
	src *SourceClient
	// cache may be nil (no Redis configured). Every read through it falls
	// through to a fresh call rather than failing.
	cache *Cache
}

func NewService(ai *gemini.Client, src *SourceClient, cache *Cache) *Service {
	return &Service{ai: ai, src: src, cache: cache}
}

// money renders integer minor units the way a Nigerian reader expects, for
// use inside prompts. Passing raw kobo to a model invites it to quote
// "10000000 naira" back at a donor.
func money(minor int64, currency string) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	major, rem := minor/100, minor%100
	// Thousands separators, built by hand: this runs inside prompt assembly
	// and does not warrant a locale package.
	s := fmt.Sprintf("%d", major)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	symbol := currency + " "
	if strings.EqualFold(currency, "NGN") {
		symbol = "₦"
	}
	return fmt.Sprintf("%s%s%s.%02d", sign, symbol, strings.Join(parts, ","), rem)
}

// snap forces a value into a known set. Gemini honours enums via the response
// schema, but a model swap or SDK change could regress, and every one of
// these values crosses a service boundary.
func snap(v string, allowed []string, fallback string) string {
	up := strings.ToUpper(strings.TrimSpace(v))
	for _, a := range allowed {
		if a == up {
			return a
		}
	}
	return fallback
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// factSheet renders the campaign context into the plain-text block every
// campaign prompt is built on.
//
// The wording is deliberate. Spend is introduced as what the organization
// "says" it spent, and the sheet states outright that CivicOS cannot verify
// it — because a model handed a list of expenses under a neutral heading will
// summarise them as established fact, and that summary is shown to donors.
func factSheet(c *Context) string {
	var b strings.Builder
	cur := c.Campaign.Currency

	fmt.Fprintf(&b, "CAMPAIGN\nTitle: %s\nSummary: %s\nCategory: %s\nStatus: %s\n",
		c.Campaign.Title, c.Campaign.Summary, c.Campaign.Category, c.Campaign.Status)
	if c.Campaign.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", c.Campaign.Description)
	}
	if c.Campaign.State != nil && c.Campaign.LGA != nil {
		fmt.Fprintf(&b, "Location: %s, %s\n", *c.Campaign.LGA, *c.Campaign.State)
	}
	if c.Campaign.IsEmergency {
		b.WriteString("This is an emergency appeal.\n")
	}
	fmt.Fprintf(&b, "Goal: %s\nRaised through CivicOS: %s from %d donors\n",
		money(c.Campaign.GoalMinor, cur), money(c.Campaign.RaisedMinor, cur), c.Campaign.DonorCount)
	if c.Campaign.EndDate != nil {
		fmt.Fprintf(&b, "Deadline: %s\n", c.Campaign.EndDate.Format("2 January 2006"))
	}

	if len(c.Milestones) > 0 {
		b.WriteString("\nWHAT THE ORGANIZATION COMMITTED TO BEFORE IT COULD PUBLISH\n")
		for _, m := range c.Milestones {
			fmt.Fprintf(&b, "- %s (%s) — %s\n", m.Title, money(m.TargetMinor, cur), m.Status)
			if m.Description != "" {
				fmt.Fprintf(&b, "  %s\n", m.Description)
			}
		}
	}

	b.WriteString("\nWHAT THE ORGANIZATION SAYS IT HAS SPENT\n")
	b.WriteString("(These are the organization's own claims. Donations settle straight to its\n")
	b.WriteString("bank account, so CivicOS never held this money and CANNOT verify any of it.)\n")
	if len(c.Spend) == 0 {
		b.WriteString("- Nothing reported yet.\n")
	} else {
		for _, s := range c.Spend {
			fmt.Fprintf(&b, "- %s: %s on %s\n", s.Description, money(s.AmountMinor, cur), s.SpentAt.Format("2 Jan 2006"))
		}
	}
	fmt.Fprintf(&b, "Total claimed as spent: %s\n", money(c.ReportedMinor(), cur))
	un := c.UnaccountedMinor()
	if un > 0 {
		fmt.Fprintf(&b, "Not yet accounted for: %s\n", money(un, cur))
	} else if un < 0 {
		fmt.Fprintf(&b, "The organization reports spending %s MORE than it raised here, which is\n"+
			"normal if it has other funding sources.\n", money(-un, cur))
	}

	if len(c.Updates) > 0 {
		b.WriteString("\nUPDATES THE ORGANIZATION HAS PUBLISHED\n")
		for _, u := range c.Updates {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", u.CreatedAt.Format("2 Jan 2006"), u.Title, u.Body)
		}
	} else {
		b.WriteString("\nUPDATES THE ORGANIZATION HAS PUBLISHED\n- None.\n")
	}

	return b.String()
}

// groundingRules are appended to every system instruction that reads real
// campaign data. Stated as prohibitions because that is what they are: the
// failure mode for all six surfaces is the model filling a gap with something
// plausible, and here the gaps are about money other people gave.
const groundingRules = `
Grounding rules, which override any instruction above:
- Use ONLY the facts given. Never invent an amount, a date, a beneficiary count, or an outcome.
- Where the organization has reported spending, describe it as what the organization REPORTS or SAYS, never as verified fact. CivicOS cannot check it.
- If the facts are thin, say so plainly and briefly. Do not pad.
- Never promise anything on the organization's behalf.
- Never state or imply that CivicOS has checked, audited, endorsed, or guaranteed anything.`
