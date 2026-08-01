package campaignai

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// ─── classify-campaign ──────────────────────────────────────────────────

type ClassifyInput struct {
	Title       string `json:"title" binding:"required,min=3"`
	Description string `json:"description" binding:"required,min=10"`
}

type ClassifyOutput struct {
	Category string `json:"category"`
	// IsEmergency is a suggestion only. Emergency appeals get extra
	// prominence in discovery, so this is a claim on other people's
	// attention — a human ticks the box.
	IsEmergency bool     `json:"isEmergency"`
	Tags        []string `json:"suggestedTags"`
	Reasoning   string   `json:"reasoning"`
	Confidence  float64  `json:"confidence"`
	Provenance
}

const classifySystem = `You are CivicAI, assisting the CivicOS civic platform in Nigeria.
Classify a fundraising campaign an organization is drafting.

Rules:
- Pick exactly one category from the allowed enum.
- Set isEmergency true ONLY for sudden events needing immediate relief: floods, fires, disease outbreaks, displacement. Long-running development work is not an emergency, however urgent it feels.
- Suggest 2 to 5 short lowercase hyphenated tags (e.g. "borehole", "flood-relief").
- Reasoning is one sentence, max 25 words.
- Confidence is 0.0 to 1.0.
- Classify only what is written. Do not infer facts that are not there.`

func (s *Service) Classify(ctx context.Context, in ClassifyInput) (*ClassifyOutput, error) {
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"category":      {Type: genai.TypeString, Enum: Categories},
			"isEmergency":   {Type: genai.TypeBoolean},
			"suggestedTags": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"reasoning":     {Type: genai.TypeString},
			"confidence":    {Type: genai.TypeNumber},
		},
		Required: []string{"category", "isEmergency", "suggestedTags", "reasoning", "confidence"},
	}
	prompt := fmt.Sprintf("Title: %s\n\nDescription: %s",
		strings.TrimSpace(in.Title), strings.TrimSpace(in.Description))

	var out ClassifyOutput
	if err := s.ai.GenerateJSON(ctx, classifySystem, prompt, schema, &out); err != nil {
		return nil, err
	}
	out.Category = snap(out.Category, Categories, "OTHER")
	out.Confidence = clamp01(out.Confidence)
	out.Provenance = s.provenance()
	return &out, nil
}

// ─── draft-campaign ─────────────────────────────────────────────────────

type DraftInput struct {
	// Brief is what the organization tells us in its own words.
	Brief string `json:"brief" binding:"required,min=20"`
	// Optional context that materially changes the draft.
	Category    string `json:"category"`
	GoalMinor   int64  `json:"goalMinor"`
	Currency    string `json:"currency"`
	State       string `json:"state"`
	LGA         string `json:"lga"`
	OrgName     string `json:"organizationName"`
	IsEmergency bool   `json:"isEmergency"`
}

// DraftMilestone is a SUGGESTED breakdown of the goal. The server rejects a
// milestone plan exceeding the goal, so these are only a starting point the
// organization edits.
type DraftMilestone struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TargetMinor int64  `json:"targetMinor"`
}

type DraftOutput struct {
	Title       string           `json:"title"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	Milestones  []DraftMilestone `json:"milestones"`
	// Warnings are things the organization should fix before submitting for
	// review — thin justification, a goal with no breakdown, claims it will
	// be asked to evidence. Cheaper to hear from a draft assistant than from
	// a rejection two days later.
	Warnings []string `json:"warnings"`
	Provenance
}

const draftSystem = `You are CivicAI, helping a verified Nigerian organization write a fundraising campaign for CivicOS.

Write in plain, direct English a secondary-school leaver can read. Nigerian context. No marketing gloss, no emotional manipulation, no urgency language beyond what the facts justify.

Rules:
- title: under 70 characters, concrete and specific. Name the place and the thing. Not "Help Our Community" — "Rebuild the Kwatarkwashi culvert".
- summary: one or two sentences, under 200 characters.
- description: 2 to 4 short paragraphs covering what the problem is, what will be done, and who benefits. Plain text, no markdown.
- milestones: 2 to 4 stages the money is spent in. Their targetMinor values MUST sum to exactly the stated goal when a goal is given; if none is given, use 0 for each and describe the stages only.
- warnings: anything the organization must supply before a reviewer will approve this — missing costings, unverifiable claims, a goal that does not match the described work. Empty array if genuinely none.

Absolute rules:
- Invent NOTHING. Every fact must come from the brief. If the brief does not say how many households benefit, do not state a number.
- Where the brief is too thin to write a section honestly, write less and add a warning saying what is missing.
- Do not promise outcomes. Describe intended work.
- Never claim CivicOS has verified, endorsed, or guaranteed anything.`

func (s *Service) Draft(ctx context.Context, in DraftInput) (*DraftOutput, error) {
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title":       {Type: genai.TypeString},
			"summary":     {Type: genai.TypeString},
			"description": {Type: genai.TypeString},
			"milestones": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"title":       {Type: genai.TypeString},
						"description": {Type: genai.TypeString},
						"targetMinor": {Type: genai.TypeInteger},
					},
					Required: []string{"title", "description", "targetMinor"},
				},
			},
			"warnings": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"title", "summary", "description", "milestones", "warnings"},
	}

	cur := in.Currency
	if cur == "" {
		cur = "NGN"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Brief from the organization:\n%s\n", strings.TrimSpace(in.Brief))
	if in.OrgName != "" {
		fmt.Fprintf(&b, "\nOrganization: %s", in.OrgName)
	}
	if in.Category != "" {
		fmt.Fprintf(&b, "\nCategory: %s", in.Category)
	}
	if in.State != "" || in.LGA != "" {
		fmt.Fprintf(&b, "\nLocation: %s %s", in.LGA, in.State)
	}
	if in.GoalMinor > 0 {
		fmt.Fprintf(&b, "\nGoal: %s (milestone targets must sum to exactly %d minor units)",
			money(in.GoalMinor, cur), in.GoalMinor)
	} else {
		b.WriteString("\nNo goal set yet — use 0 for every milestone target.")
	}
	if in.IsEmergency {
		b.WriteString("\nThe organization has marked this an emergency appeal.")
	}

	var out DraftOutput
	if err := s.ai.GenerateJSONTimeout(ctx, deliberateTimeout, draftSystem, b.String(), schema, &out); err != nil {
		return nil, err
	}

	// The model is told to make the milestones sum to the goal. When it does
	// not, say so rather than silently rewriting the numbers: the split is
	// the organization's decision about its own money, and a corrected total
	// that nobody chose is worse than an honest warning.
	if in.GoalMinor > 0 {
		var sum int64
		for _, m := range out.Milestones {
			sum += m.TargetMinor
		}
		if sum != in.GoalMinor {
			out.Warnings = append(out.Warnings, fmt.Sprintf(
				"The suggested milestones add up to %s, not the %s goal. Adjust them before submitting — the server will reject a plan that exceeds the goal.",
				money(sum, cur), money(in.GoalMinor, cur)))
		}
	}
	out.Provenance = s.provenance()
	return &out, nil
}
