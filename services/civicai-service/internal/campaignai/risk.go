package campaignai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

// assess-campaign-risk is the endpoint that needed the most care, so the
// reasoning is written down rather than left in a commit message.
//
// It produces a fraud signal about a named organization asking the public for
// money — often for a flood, a clinic, a school. The funding plan is explicit
// about what that means: "an AI that can block fundraising is an AI that can
// be wrong about someone's flood relief". So:
//
//  1. It is PLATFORM_ADMIN only. Not org admins — an organization must never
//     be able to run a fraud probe on a rival. Enforced in handler.go.
//  2. It never writes. Nothing here sets Campaign.RiskScore, changes a
//     status, or notifies anyone. The output exists to be read by a person
//     who then decides, using the ordinary review and pause paths that
//     already have their own audit trails.
//  3. It is asked for OBSERVATIONS, not a verdict. The model is told, in the
//     system instruction, that it is not deciding anything — because a model
//     asked to judge produces judgments, and a reviewer reading "LIKELY
//     FRAUD" is anchored before they have opened the campaign.
//  4. Every signal must cite the fact it rests on. A concern a reviewer
//     cannot trace back to something concrete is noise that costs a real
//     organization real money while someone investigates it.
//
// What it is genuinely useful for: a reviewer with forty campaigns in a queue
// wants to know which three to open first. That is the whole job.

type RiskInput struct {
	CampaignID string `json:"campaignId" binding:"required,uuid"`
}

// RiskBands are deliberately coarse and phrased as review priorities rather
// than as accusations. There is no "FRAUDULENT" band because the model is in
// no position to conclude that and the word would follow the organization
// around the admin console.
var RiskBands = []string{"ROUTINE", "WORTH_A_LOOK", "REVIEW_CLOSELY"}

// RiskSignal is one observation, tied to the fact it came from.
type RiskSignal struct {
	// Concern in one plain sentence.
	Concern string `json:"concern"`
	// Evidence quotes or points at the specific campaign fact behind it. The
	// schema requires it: a signal without evidence does not get to exist.
	Evidence string `json:"evidence"`
	// InnocentExplanation is the most likely benign reading of the same
	// fact. Required for the same reason as evidence — most campaigns that
	// look odd are run by people who are bad at paperwork, not thieves, and
	// a reviewer should see both readings at once.
	InnocentExplanation string `json:"innocentExplanation"`
}

type RiskOutput struct {
	Band    string       `json:"band"`
	Signals []RiskSignal `json:"signals"`
	// WhatToCheck is what a human should actually do next — concrete,
	// checkable steps, not "investigate further".
	WhatToCheck []string `json:"whatToCheck"`
	Confidence  float64  `json:"confidence"`
	// Disclaimer travels with the payload so it cannot be dropped by a
	// client that renders only the fields it cares about.
	Disclaimer string `json:"disclaimer"`
	Provenance
}

const riskDisclaimer = "Advisory only. This is a reading of published campaign data by a language model, " +
	"not a finding. It has not checked any bank record, document, or identity, and it is often wrong about " +
	"organizations that are simply bad at paperwork. Nothing has been changed on the campaign. Decide for yourself."

const riskSystem = `You are CivicAI, helping a CivicOS platform administrator decide which campaigns in a review queue to open first.

You are NOT deciding anything. You do not approve, reject, pause, or flag. A person reads what you write and makes every decision. Write accordingly: observations a reviewer can check, not conclusions.

The organizations here are mostly small Nigerian NGOs and community groups raising money for real problems — floods, boreholes, school repairs, clinics. Most oddities in their paperwork are inexperience, not dishonesty. A false alarm costs a genuine organization donations and reputation while someone investigates it. Weigh that.

Rules:
- band: ROUTINE (nothing stands out), WORTH_A_LOOK (one or two things a reviewer should confirm), REVIEW_CLOSELY (several things that together justify opening this one first). Use REVIEW_CLOSELY sparingly.
- signals: up to 5. Each needs a concern, the specific evidence from the campaign data it rests on, and the most likely innocent explanation for that same evidence. If you cannot point at concrete evidence, do not raise the signal.
- whatToCheck: up to 4 concrete things a human can verify — a document to request, a figure to reconcile, a question to put to the organization. Not "investigate further".
- confidence: 0.0 to 1.0, in your reading of the data. Be honest when the data is thin; thin data is the normal case for a new campaign and is not itself suspicious.

Things that are legitimately worth noticing: a goal far out of proportion to the described work; milestones that do not add up to what the money is for; a description that is vague about who benefits; money raised long ago with nothing reported since; reported spending that does not match the stated plan; urgency language on something that is not an emergency; text that appears copied from elsewhere.

Things that are NOT signals on their own: a small organization; a new organization; simple or ungrammatical writing; a large goal for genuinely expensive work; an emergency appeal moving fast; no spending reported by a campaign that has only just published.` + groundingRules

func (s *Service) AssessRisk(ctx context.Context, campaignID, bearer string) (*RiskOutput, error) {
	// Authorise first, then consult the cache — see Impact for why.
	cc, err := s.src.Fetch(ctx, campaignID, bearer)
	if err != nil {
		return nil, err
	}
	var cached RiskOutput
	if s.cache.get(ctx, riskKey(campaignID), &cached) {
		return &cached, nil
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"band": {Type: genai.TypeString, Enum: RiskBands},
			"signals": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"concern":             {Type: genai.TypeString},
						"evidence":            {Type: genai.TypeString},
						"innocentExplanation": {Type: genai.TypeString},
					},
					// All three required: the schema is what stops a bare
					// accusation with nothing behind it reaching a reviewer.
					Required: []string{"concern", "evidence", "innocentExplanation"},
				},
			},
			"whatToCheck": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"confidence":  {Type: genai.TypeNumber},
		},
		Required: []string{"band", "signals", "whatToCheck", "confidence"},
	}

	var b strings.Builder
	b.WriteString(factSheet(cc))
	// How long it has been live, stated outright. One of the signals the
	// prompt asks for is money raised a while ago with nothing reported
	// since, and a model given only two dates reasons about elapsed time
	// unreliably. Silence is only meaningful against a duration.
	if cc.Campaign.PublishedAt != nil {
		days := int(time.Since(*cc.Campaign.PublishedAt).Hours() / 24)
		fmt.Fprintf(&b, "\nPublished %s — %d day(s) ago.\n",
			cc.Campaign.PublishedAt.Format("2 January 2006"), days)
	} else {
		b.WriteString("\nNot yet published.\n")
	}
	fmt.Fprintf(&b, "The organization has published %d update(s) and %d spending record(s).\n",
		len(cc.Updates), len(cc.Spend))

	var out RiskOutput
	if err := s.ai.GenerateJSONTimeout(ctx, deliberateTimeout, riskSystem, b.String(), schema, &out); err != nil {
		return nil, err
	}

	out.Band = snap(out.Band, RiskBands, "ROUTINE")
	out.Confidence = clamp01(out.Confidence)

	// Drop any signal that arrived without evidence. The schema requires the
	// field, but "required" only means present — a model can satisfy it with
	// an empty string, and an unevidenced concern about a real organization
	// is exactly what must not reach a reviewer.
	kept := out.Signals[:0]
	for _, sig := range out.Signals {
		if strings.TrimSpace(sig.Evidence) != "" && strings.TrimSpace(sig.Concern) != "" {
			kept = append(kept, sig)
		}
	}
	out.Signals = kept
	// A band that outran its own evidence is downgraded rather than shown.
	if len(out.Signals) == 0 {
		out.Band = "ROUTINE"
	}

	out.Disclaimer = riskDisclaimer
	out.Provenance = s.provenance()
	s.cache.set(ctx, riskKey(campaignID), out, riskTTL)
	return &out, nil
}
