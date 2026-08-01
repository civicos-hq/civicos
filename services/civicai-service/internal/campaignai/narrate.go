package campaignai

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// The three surfaces that read a real campaign: an impact summary for the
// public, a donor update, and a completion report. All three write ABOUT
// money other people gave, from claims CivicOS cannot verify, so all three
// carry groundingRules and none of them writes anything back.

// ─── summarize-campaign-impact ──────────────────────────────────────────

type ImpactInput struct {
	CampaignID string `json:"campaignId" binding:"required,uuid"`
}

type ImpactOutput struct {
	Summary string `json:"summary"`
	// Highlights are what actually happened, one line each — drawn only from
	// reported spend and published updates.
	Highlights []string `json:"highlights"`
	// Gaps is the honest other half: what a reader cannot tell from what has
	// been published. A summary of a campaign with no updates and no spend
	// records should say so rather than producing warm sentences about
	// nothing.
	Gaps []string `json:"gaps"`
	Provenance
}

const impactSystem = `You are CivicAI, writing a short public summary of what a fundraising campaign has achieved so far, for citizens reading the campaign page on CivicOS.

Audience: ordinary Nigerians, including donors deciding whether this organization can be trusted with more money. Plain English, calm, factual, no marketing tone.

Rules:
- summary: 2 to 4 sentences on where the campaign stands. Mention money raised and what the organization reports doing with it.
- highlights: up to 4 concrete things that have happened, one short line each. Only from reported spend and published updates. Empty array if nothing concrete has been reported.
- gaps: up to 4 things a reader genuinely cannot tell from what has been published — money raised but not accounted for, milestones with no progress reported, long silences. Empty array only if there are truly none.

Be even-handed. This is not promotional copy and it is not an accusation. A campaign that raised money and reported nothing should read as exactly that, without insinuation.` + groundingRules

func (s *Service) Impact(ctx context.Context, campaignID, bearer string) (*ImpactOutput, error) {
	// The source fetch stays OUTSIDE the cache check on purpose: it is what
	// authorises the caller. Serving a cached summary without it would let
	// anyone who knows a campaign id read one, regardless of access.
	cc, err := s.src.Fetch(ctx, campaignID, bearer)
	if err != nil {
		return nil, err
	}
	var cached ImpactOutput
	if s.cache.get(ctx, impactKey(campaignID), &cached) {
		return &cached, nil
	}
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"summary":    {Type: genai.TypeString},
			"highlights": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"gaps":       {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"summary", "highlights", "gaps"},
	}
	var out ImpactOutput
	if err := s.ai.GenerateJSONTimeout(ctx, deliberateTimeout, impactSystem, factSheet(cc), schema, &out); err != nil {
		return nil, err
	}
	out.Provenance = s.provenance()
	// generatedAt is cached with the payload, so a reader can always see how
	// old the summary they are looking at actually is.
	s.cache.set(ctx, impactKey(campaignID), out, impactTTL)
	return &out, nil
}

// ─── draft-donor-update ─────────────────────────────────────────────────

type DonorUpdateInput struct {
	CampaignID string `json:"campaignId" binding:"required,uuid"`
	// Brief is what the organization wants to tell donors, in its own words.
	// Required: an update generated purely from the ledger would be the
	// platform speaking in the organization's voice about work only the
	// organization witnessed.
	Brief string `json:"brief" binding:"required,min=10"`
}

type DonorUpdateOutput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// Warnings flag anything in the brief the organization is about to
	// assert to people who gave it money and cannot check — a claimed
	// outcome with no corresponding spend record, for instance.
	Warnings []string `json:"warnings"`
	Provenance
}

const donorUpdateSystem = `You are CivicAI, helping a Nigerian organization write an update for the people who donated to its campaign on CivicOS.

Audience: donors. They gave real money and are owed a straight account. Write as the organization ("we"), in plain English, warm but not effusive. No marketing language. No fundraising ask unless the brief contains one.

Rules:
- title: under 60 characters, says what happened.
- body: 2 to 4 short paragraphs. Plain text, no markdown. Lead with what has been done, then what is next, then anything that has gone wrong or been delayed — donors find out eventually and hearing it here is better.
- warnings: things the organization is claiming that the published record does not support, or that donors will reasonably ask to see evidence for. Empty array if none.

The brief is the organization's account of what happened. Use it, but do not embellish it.` + groundingRules

func (s *Service) DonorUpdate(ctx context.Context, in DonorUpdateInput, bearer string) (*DonorUpdateOutput, error) {
	cc, err := s.src.Fetch(ctx, in.CampaignID, bearer)
	if err != nil {
		return nil, err
	}
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title":    {Type: genai.TypeString},
			"body":     {Type: genai.TypeString},
			"warnings": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"title", "body", "warnings"},
	}
	prompt := fmt.Sprintf("%s\n\nWHAT THE ORGANIZATION WANTS TO TELL DONORS\n%s",
		factSheet(cc), strings.TrimSpace(in.Brief))

	var out DonorUpdateOutput
	if err := s.ai.GenerateJSONTimeout(ctx, deliberateTimeout, donorUpdateSystem, prompt, schema, &out); err != nil {
		return nil, err
	}
	out.Provenance = s.provenance()
	return &out, nil
}

// ─── draft-completion-report ────────────────────────────────────────────

type CompletionReportInput struct {
	CampaignID string `json:"campaignId" binding:"required,uuid"`
	// Brief is the organization's account of how the work finished.
	Brief string `json:"brief" binding:"required,min=20"`
}

type CompletionReportOutput struct {
	Body string `json:"body"`
	// UnaccountedMinor is computed here, NOT by the model. Arithmetic about
	// unexplained money is the one number on this page nobody should be
	// generating — it is frozen into the public record at filing time and a
	// hallucinated figure would be indistinguishable from a true one.
	UnaccountedMinor int64  `json:"unaccountedMinor"`
	Currency         string `json:"currency"`
	// MustExplain is set when money is unaccounted for. The report will be
	// published with that shortfall named permanently, so the drafting step
	// is the right moment to say so.
	MustExplain bool     `json:"mustExplain"`
	Warnings    []string `json:"warnings"`
	Provenance
}

const completionSystem = `You are CivicAI, helping a Nigerian organization write the closing report for a fundraising campaign on CivicOS. This is the permanent public record of what the money achieved.

Audience: donors and the wider community. Plain English, first person plural, factual.

Rules:
- body: 3 to 5 short paragraphs. Plain text, no markdown. Cover what was done against each milestone, what the money went on, what was NOT achieved, and what happens to anything left over.
- Where money the campaign raised has not been accounted for, the report MUST address it directly. Do not skip it, soften it, or bury it. State the amount and what happened to it.
- warnings: anything the organization should add before publishing — a milestone never mentioned, a claim with no spend record behind it, an unexplained balance. Empty array if none.

This report is published with the shortfall recorded at the moment of filing, and that figure does not change afterwards. A thin report cannot be fixed later.` + groundingRules

func (s *Service) CompletionReport(ctx context.Context, in CompletionReportInput, bearer string) (*CompletionReportOutput, error) {
	cc, err := s.src.Fetch(ctx, in.CampaignID, bearer)
	if err != nil {
		return nil, err
	}
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"body":     {Type: genai.TypeString},
			"warnings": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
		Required: []string{"body", "warnings"},
	}

	un := cc.UnaccountedMinor()
	var b strings.Builder
	b.WriteString(factSheet(cc))
	fmt.Fprintf(&b, "\nHOW THE ORGANIZATION SAYS THE WORK FINISHED\n%s\n", strings.TrimSpace(in.Brief))
	if un > 0 {
		fmt.Fprintf(&b, "\nIMPORTANT: %s of what was raised has NOT been accounted for. "+
			"The report must address this directly.\n", money(un, cc.Campaign.Currency))
	}

	var out CompletionReportOutput
	if err := s.ai.GenerateJSONTimeout(ctx, deliberateTimeout, completionSystem, b.String(), schema, &out); err != nil {
		return nil, err
	}
	out.UnaccountedMinor = un
	out.Currency = cc.Campaign.Currency
	out.MustExplain = un > 0
	if un > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"%s is still unaccounted for. Filing now records that figure permanently on the public page, even if you report more spending later.",
			money(un, cc.Campaign.Currency)))
	}
	out.Provenance = s.provenance()
	return &out, nil
}
