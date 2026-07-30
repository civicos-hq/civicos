package domain

import (
	"strings"
	"time"
)

type OrgKind string
type OrgJurisdiction string
type OrgMemberRole string
type AnnouncementStatus string
type ProjectStatus string
type AssignmentStatus string
type ConsultationStatus string
type QuestionType string
type CampaignStatus string
type CampaignCategory string
type MilestoneStatus string
type PauseReason string

const (
	OrgKindGovernment OrgKind = "GOVERNMENT"
	OrgKindAgency     OrgKind = "AGENCY"
	OrgKindNGO        OrgKind = "NGO"
	OrgKindUtility    OrgKind = "UTILITY"
	OrgKindOther      OrgKind = "OTHER"

	JurisdictionNational  OrgJurisdiction = "NATIONAL"
	JurisdictionState     OrgJurisdiction = "STATE"
	JurisdictionLGA       OrgJurisdiction = "LGA"
	JurisdictionCommunity OrgJurisdiction = "COMMUNITY"

	MemberRoleOwner OrgMemberRole = "OWNER"
	MemberRoleAdmin OrgMemberRole = "ADMIN"
	MemberRoleStaff OrgMemberRole = "STAFF"

	AnnouncementDraft     AnnouncementStatus = "DRAFT"
	AnnouncementPublished AnnouncementStatus = "PUBLISHED"
	AnnouncementArchived  AnnouncementStatus = "ARCHIVED"

	ProjectPlanned   ProjectStatus = "PLANNED"
	ProjectActive    ProjectStatus = "ACTIVE"
	ProjectPaused    ProjectStatus = "PAUSED"
	ProjectCompleted ProjectStatus = "COMPLETED"
	ProjectCancelled ProjectStatus = "CANCELLED"

	AssignmentReceived   AssignmentStatus = "RECEIVED"
	AssignmentInProgress AssignmentStatus = "IN_PROGRESS"
	AssignmentCompleted  AssignmentStatus = "COMPLETED"
	AssignmentRejected   AssignmentStatus = "REJECTED"

	ConsultationDraft     ConsultationStatus = "DRAFT"
	ConsultationPublished ConsultationStatus = "PUBLISHED"
	ConsultationClosed    ConsultationStatus = "CLOSED"

	QuestionShortText    QuestionType = "SHORT_TEXT"
	QuestionLongText     QuestionType = "LONG_TEXT"
	QuestionSingleChoice QuestionType = "SINGLE_CHOICE"
	QuestionMultiChoice  QuestionType = "MULTI_CHOICE"
	QuestionYesNo        QuestionType = "YES_NO"

	// Campaign lifecycle. The spec's diagram (docs/product/CivicOS Community
	// Funding Feature.pdf) runs Draft → Verification Review → Approved →
	// Published → Receiving Donations → … → Archived. Two departures, both
	// reasoned in docs/product/community-funding-plan.md:
	//
	//   1. The spec's diagram has no failure states, but its own Governance
	//      section requires pausing a campaign for fraud, expired
	//      verification, misuse, suspension, or false information. So
	//      NEEDS_CHANGES, REJECTED and PAUSED exist.
	//   2. "Receiving Donations" is not a status — it is what PUBLISHED
	//      means. Modelling it separately would let a campaign be published
	//      but not fundable with no way to express the difference.
	//
	// FUNDED means the goal was reached while donations may still arrive;
	// COMPLETED means the work is done; REPORTED means the final report is
	// published, which is the gate for ARCHIVED.
	CampaignDraft         CampaignStatus = "DRAFT"
	CampaignPendingReview CampaignStatus = "PENDING_REVIEW"
	CampaignNeedsChanges  CampaignStatus = "NEEDS_CHANGES"
	CampaignApproved      CampaignStatus = "APPROVED"
	CampaignPublished     CampaignStatus = "PUBLISHED"
	CampaignFunded        CampaignStatus = "FUNDED"
	CampaignPaused        CampaignStatus = "PAUSED"
	CampaignRejected      CampaignStatus = "REJECTED"
	CampaignCompleted     CampaignStatus = "COMPLETED"
	CampaignReported      CampaignStatus = "REPORTED"
	CampaignArchived      CampaignStatus = "ARCHIVED"

	// Campaign categories — the seven groups from the spec. Stored as the
	// group, not the leaf: the spec lists examples under each heading
	// ("Flood", "Fire", … under Emergency Relief) which are description
	// detail, not taxonomy. Keeping the enum at group level means adding a
	// flood type never needs a migration.
	CategoryEmergencyRelief      CampaignCategory = "EMERGENCY_RELIEF"
	CategoryCommunityDevelopment CampaignCategory = "COMMUNITY_DEVELOPMENT"
	CategoryEducation            CampaignCategory = "EDUCATION"
	CategoryHealthcare           CampaignCategory = "HEALTHCARE"
	CategoryEnvironment          CampaignCategory = "ENVIRONMENT"
	CategoryAgriculture          CampaignCategory = "AGRICULTURE"
	CategoryOther                CampaignCategory = "OTHER"

	MilestonePlanned    MilestoneStatus = "PLANNED"
	MilestoneInProgress MilestoneStatus = "IN_PROGRESS"
	MilestoneCompleted  MilestoneStatus = "COMPLETED"

	// Pause reasons — the five Governance triggers from the spec, plus
	// OTHER. A code rather than free text because these are the grounds on
	// which a live fundraiser is stopped: they need to be countable
	// (how often do we pause for fraud?), filterable in the admin queue,
	// and consistent across reviewers. The free-text note is retained
	// alongside for the specifics.
	PauseFraudDetected         PauseReason = "FRAUD_DETECTED"
	PauseVerificationExpired   PauseReason = "VERIFICATION_EXPIRED"
	PauseMisuseReported        PauseReason = "MISUSE_REPORTED"
	PauseOrganizationSuspended PauseReason = "ORGANIZATION_SUSPENDED"
	PauseFalseInformation      PauseReason = "FALSE_INFORMATION"
	PauseOther                 PauseReason = "OTHER"
)

// ValidPauseReason reports whether v is a known pause reason code.
func ValidPauseReason(v string) bool {
	switch PauseReason(v) {
	case PauseFraudDetected, PauseVerificationExpired, PauseMisuseReported,
		PauseOrganizationSuspended, PauseFalseInformation, PauseOther:
		return true
	}
	return false
}

type Organization struct {
	ID                string          `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string          `gorm:"not null" json:"name"`
	Slug              string          `gorm:"uniqueIndex;not null" json:"slug"`
	Kind              OrgKind         `gorm:"type:varchar(30);default:'OTHER'" json:"kind"`
	Jurisdiction      OrgJurisdiction `gorm:"type:varchar(30);default:'COMMUNITY'" json:"jurisdiction"`
	State             *string         `json:"state,omitempty"`
	LGA               *string         `json:"lga,omitempty"`
	Description       *string         `json:"description,omitempty"`
	LogoURL           *string         `json:"logoUrl,omitempty"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Website           *string         `json:"website,omitempty"`
	Verified          bool            `gorm:"default:false" json:"verified"`
	MemberCount       int             `gorm:"default:0" json:"memberCount"`
	AnnouncementCount int             `gorm:"default:0" json:"announcementCount"`
	ProjectCount      int             `gorm:"default:0" json:"projectCount"`
	AssignmentCount   int             `gorm:"default:0" json:"assignmentCount"`

	// ─── Funding verification (Community Funding, Phase 2) ───
	//
	// The product spec requires an organization to provide registration
	// number, country, official email, a named representative, identity
	// verification, supporting documents and bank-account verification
	// before it may raise money.
	//
	// These are deliberately SEPARATE from the existing `Verified` flag.
	// `Verified` is the general "this org is who it says it is" badge that
	// predates funding and is already set on live orgs; tightening its
	// meaning would retroactively unverify every existing organization.
	// Instead, raising money requires `Verified` AND the funding fields —
	// see FundingEligible below.
	RegistrationNumber *string `json:"registrationNumber,omitempty"`
	Country            *string `json:"country,omitempty"`
	OfficialEmail      *string `json:"officialEmail,omitempty"`
	// RepresentativeName is the human accountable for the org's campaigns.
	// The spec calls this "Representative"; named explicitly here so it is
	// not confused with the platform's REPRESENTATIVE user role.
	RepresentativeName *string `json:"representativeName,omitempty"`
	// BankAccountVerified is set by a platform admin after checking the
	// settlement account out-of-band. Phase 5 pays out to it. It is a bool,
	// not account details: CivicOS should not hold bank numbers it has no
	// use for until the payout integration exists.
	BankAccountVerified   bool       `gorm:"default:false" json:"bankAccountVerified"`
	SupportingDocumentURL *string    `json:"supportingDocumentUrl,omitempty"`
	FundingVerifiedAt     *time.Time `json:"fundingVerifiedAt,omitempty"`
	FundingVerifiedByID   *string    `gorm:"type:uuid" json:"fundingVerifiedById,omitempty"`

	CreatedByID string    `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// FundingEligible reports whether this organization may raise money.
//
// Checked at campaign-submit time rather than at campaign-create time, so an
// org can draft while its paperwork is still being processed. Returns the
// list of missing requirements alongside the verdict so the API can tell the
// org exactly what to supply — "your organization is not eligible" with no
// further detail is not an actionable error.
func (o *Organization) FundingEligible() (bool, []string) {
	var missing []string
	blank := func(p *string) bool { return p == nil || strings.TrimSpace(*p) == "" }

	if !o.Verified {
		missing = append(missing, "organization verification")
	}
	if blank(o.RegistrationNumber) {
		missing = append(missing, "registration number")
	}
	if blank(o.Country) {
		missing = append(missing, "country")
	}
	if blank(o.OfficialEmail) {
		missing = append(missing, "official email")
	}
	if blank(o.RepresentativeName) {
		missing = append(missing, "named representative")
	}
	if !o.BankAccountVerified {
		missing = append(missing, "bank account verification")
	}
	return len(missing) == 0, missing
}

type OrgMember struct {
	ID             string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string        `gorm:"type:uuid;not null;uniqueIndex:idx_org_user" json:"organizationId"`
	UserID         string        `gorm:"type:uuid;not null;uniqueIndex:idx_org_user" json:"userId"`
	UserName       string        `gorm:"not null" json:"userName"`
	UserRole       string        `gorm:"not null" json:"userRole"`
	Role           OrgMemberRole `gorm:"type:varchar(20);default:'STAFF'" json:"role"`
	JoinedAt       time.Time     `json:"joinedAt"`
}

type Announcement struct {
	ID             string             `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string             `gorm:"type:uuid;not null;index" json:"organizationId"`
	Title          string             `gorm:"not null" json:"title"`
	Body           string             `gorm:"not null" json:"body"`
	Status         AnnouncementStatus `gorm:"type:varchar(20);default:'DRAFT'" json:"status"`
	PublishedAt    *time.Time         `json:"publishedAt,omitempty"`
	AuthorID       string             `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName     string             `gorm:"not null" json:"authorName"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

type Project struct {
	ID              string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID  string        `gorm:"type:uuid;not null;index" json:"organizationId"`
	Title           string        `gorm:"not null" json:"title"`
	Description     string        `gorm:"not null" json:"description"`
	Status          ProjectStatus `gorm:"type:varchar(20);default:'PLANNED'" json:"status"`
	StartDate       *time.Time    `json:"startDate,omitempty"`
	ExpectedEndDate *time.Time    `json:"expectedEndDate,omitempty"`
	BudgetKobo      *int64        `json:"budgetKobo,omitempty"`
	CommunityID     *string       `gorm:"type:uuid;index" json:"communityId,omitempty"`
	CreatedByID     string        `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// IssueAssignment records that an org has taken responsibility for an
// externally-owned Issue (owned by community-service). The IssueID is a
// bare UUID reference — this service does not FK across the boundary.
type IssueAssignment struct {
	ID             string           `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string           `gorm:"type:uuid;not null;uniqueIndex:idx_org_issue" json:"organizationId"`
	IssueID        string           `gorm:"type:uuid;not null;uniqueIndex:idx_org_issue" json:"issueId"`
	Status         AssignmentStatus `gorm:"type:varchar(20);default:'RECEIVED'" json:"status"`
	Note           *string          `json:"note,omitempty"`
	AssignedByID   string           `gorm:"type:uuid;not null" json:"assignedById"`
	AssignedByName string           `gorm:"not null" json:"assignedByName"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

// ProgressUpdate is the "respond publicly" primitive. Every update belongs
// to an org and points at either an Issue (an assigned report) or a
// Project. Public updates are readable by anyone; internal notes are
// member-only.
type ProgressUpdate struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string    `gorm:"type:uuid;not null;index" json:"organizationId"`
	IssueID        *string   `gorm:"type:uuid;index" json:"issueId,omitempty"`
	ProjectID      *string   `gorm:"type:uuid;index" json:"projectId,omitempty"`
	Body           string    `gorm:"not null" json:"body"`
	IsPublic       bool      `gorm:"default:true" json:"isPublic"`
	AuthorID       string    `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName     string    `gorm:"not null" json:"authorName"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Consultation is a structured feedback ask published by an organization to
// either the whole org membership or a single community. Lifecycle is
// DRAFT → PUBLISHED → CLOSED. Editing questions is only allowed while the
// consultation is still a DRAFT — once published, the form is frozen so
// early responders and late responders are answering the same questions.
//
// CommunityID is stored but NOT enforced on response submission: a
// consultation "aimed at" one community is an audience signal, and any
// verified user can respond. This matches the platform-wide accountability
// principle that participation is deliberate and identified.
type Consultation struct {
	ID             string             `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string             `gorm:"type:uuid;not null;index" json:"organizationId"`
	CommunityID    *string            `gorm:"type:uuid;index" json:"communityId,omitempty"`
	Title          string             `gorm:"not null" json:"title"`
	Summary        string             `gorm:"not null" json:"summary"`
	Description    string             `gorm:"type:text;not null" json:"description"`
	CoverImageURL  *string            `json:"coverImageUrl,omitempty"`
	Status         ConsultationStatus `gorm:"type:varchar(20);default:'DRAFT';index" json:"status"`
	OpensAt        *time.Time         `json:"opensAt,omitempty"`
	ClosesAt       *time.Time         `json:"closesAt,omitempty"`
	ResponseCount  int                `gorm:"default:0" json:"responseCount"`
	AuthorID       string             `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName     string             `gorm:"not null" json:"authorName"`
	PublishedAt    *time.Time         `json:"publishedAt,omitempty"`
	ClosedAt       *time.Time         `json:"closedAt,omitempty"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

// Question belongs to a Consultation. `Position` orders questions in the
// form; the client is responsible for continuous, gap-free values but the
// server sorts by Position anyway. `Options` is a JSON array used by the
// two choice types; empty for text/yes-no.
type ConsultationQuestion struct {
	ID             string       `gorm:"type:uuid;primaryKey" json:"id"`
	ConsultationID string       `gorm:"type:uuid;not null;index" json:"consultationId"`
	Position       int          `gorm:"not null" json:"position"`
	Prompt         string       `gorm:"not null" json:"prompt"`
	HelpText       *string      `json:"helpText,omitempty"`
	Type           QuestionType `gorm:"type:varchar(20);not null" json:"type"`
	// Options is `[]` for question types that don't use them (SHORT_TEXT,
	// LONG_TEXT, YES_NO) and the choice list for SINGLE_CHOICE +
	// MULTI_CHOICE. The service's normalizeOptions() always writes an
	// empty slice — never nil — so the column carries JSON `[]`, never
	// SQL NULL. The `not null default '[]'` schema tag reinforces that
	// contract at the DB layer for any fresh AutoMigrate.
	Options   []string  `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"options"`
	Required  bool      `gorm:"default:false" json:"required"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConsultationResponse is a citizen's submitted response set. Compound
// unique index (consultation_id, user_id) enforces one submission per
// verified user. Answers are child rows.
type ConsultationResponse struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	ConsultationID string    `gorm:"type:uuid;not null;uniqueIndex:idx_consultation_respondent;index" json:"consultationId"`
	UserID         string    `gorm:"type:uuid;not null;uniqueIndex:idx_consultation_respondent" json:"userId"`
	SubmittedAt    time.Time `gorm:"not null;index" json:"submittedAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ConsultationAnswer stores a single question's answer inside a response.
// Exactly one of TextValue/Selections carries data — TextValue for
// SHORT_TEXT and LONG_TEXT; Selections for SINGLE_CHOICE, MULTI_CHOICE,
// and YES_NO (encoded as ["YES"] or ["NO"] for consistency).
type ConsultationAnswer struct {
	ID         string   `gorm:"type:uuid;primaryKey" json:"id"`
	ResponseID string   `gorm:"type:uuid;not null;uniqueIndex:idx_answer_response_question;index" json:"responseId"`
	QuestionID string   `gorm:"type:uuid;not null;uniqueIndex:idx_answer_response_question;index" json:"questionId"`
	TextValue  *string  `json:"textValue,omitempty"`
	Selections []string `gorm:"type:jsonb;serializer:json" json:"selections,omitempty"`
}

// ConsultationOutcome is the "close the loop" primitive — after a
// consultation closes, the org publishes a summary of findings and what
// happens next. Exactly one outcome per consultation (unique index).
type ConsultationOutcome struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	ConsultationID string    `gorm:"type:uuid;not null;uniqueIndex" json:"consultationId"`
	Summary        string    `gorm:"type:text;not null" json:"summary"`
	Decisions      string    `gorm:"type:text;not null" json:"decisions"`
	NextSteps      string    `gorm:"type:text;not null" json:"nextSteps"`
	AuthorID       string    `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName     string    `gorm:"not null" json:"authorName"`
	PublishedAt    time.Time `gorm:"not null" json:"publishedAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ─── Community Funding ──────────────────────────────────────────────────
// See docs/product/community-funding-plan.md. Phase 1 covers the campaign
// and its milestones only — no Donation/Withdrawal ledger yet, and no
// payment code anywhere in this service.

// Campaign is the fundable thing. It belongs to an Organization, and that
// organization must be Verified before the campaign can leave review.
//
// Money is stored as integer minor units (kobo for NGN, pence for GBP) in
// int64, following the Project.BudgetKobo precedent. Never float — binary
// floating point cannot represent 0.01 exactly, and rounding drift on money
// is a defect, not an inconvenience. Currency is explicit per campaign
// rather than assumed: the spec's worked example is in £ while the rest of
// this codebase is Nigeria-first, so an implicit currency would be a bug
// waiting to happen.
type Campaign struct {
	ID             string           `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string           `gorm:"type:uuid;not null;index" json:"organizationId"`
	Title          string           `gorm:"not null" json:"title"`
	Slug           string           `gorm:"uniqueIndex;not null" json:"slug"`
	Summary        string           `gorm:"not null" json:"summary"`
	Description    string           `gorm:"type:text;not null" json:"description"`
	Category       CampaignCategory `gorm:"type:varchar(30);not null;index" json:"category"`
	Status         CampaignStatus   `gorm:"type:varchar(20);default:'DRAFT';index" json:"status"`
	CoverImageURL  *string          `json:"coverImageUrl,omitempty"`

	// Currency is ISO-4217 (e.g. "NGN"). Immutable after creation — a
	// campaign that has taken money in one currency cannot be reinterpreted
	// in another, and Phase 1 forbidding the edit means Phase 3 inherits
	// that guarantee for free.
	Currency  string `gorm:"type:varchar(3);not null;default:'NGN'" json:"currency"`
	GoalMinor int64  `gorm:"not null" json:"goalMinor"`

	// RaisedMinor and DonorCount are a cached PROJECTION of the donation
	// ledger, not the source of truth. Phase 1 never writes anything but
	// zero here. When Phase 3 lands, these are recomputed from settled
	// donations inside the same transaction as the ledger write — never
	// incremented in place, because a replayed webhook would inflate them.
	RaisedMinor int64 `gorm:"not null;default:0" json:"raisedMinor"`
	DonorCount  int   `gorm:"not null;default:0" json:"donorCount"`

	// Where the campaign applies. CommunityID and ProjectID are bare UUID
	// references, not FKs: Community is owned by identity-service and this
	// service does not FK across a service boundary (same rule as
	// IssueAssignment.IssueID).
	CommunityID *string `gorm:"type:uuid;index" json:"communityId,omitempty"`
	ProjectID   *string `gorm:"type:uuid;index" json:"projectId,omitempty"`
	State       *string `json:"state,omitempty"`
	LGA         *string `json:"lga,omitempty"`

	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`

	// IsEmergency flags the spec's "Approve emergency campaigns" admin
	// capability. It does not relax any verification requirement — it only
	// marks the campaign for a faster review queue.
	IsEmergency bool `gorm:"default:false;index" json:"isEmergency"`

	// RiskScore is CivicAI's advisory fraud signal (Phase 6), 0-100.
	// json:"-" because it is reviewer-only: exposing "this campaign scored
	// 71" publicly would defame legitimate organizations on the strength of
	// a model's guess. Admin surfaces read it off the record directly.
	RiskScore *int `json:"-"`

	// Review trail. ApprovalStatus deliberately reuses identity-service's
	// vocabulary (NONE/PENDING/APPROVED/NEEDS_CHANGES/REJECTED) so the admin
	// console can render campaign review with the same components it already
	// uses for representative and organization applications.
	ApprovalStatus string     `gorm:"type:varchar(20);default:'NONE';index" json:"approvalStatus"`
	ReviewedByID   *string    `gorm:"type:uuid" json:"reviewedById,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	ReviewNote     *string    `json:"reviewNote,omitempty"`

	// Pause detail, set when Status is PAUSED. The code is one of the
	// spec's Governance triggers (see PauseReason); the note carries the
	// specifics for the organization and the audit trail.
	PauseReasonCode *PauseReason `gorm:"type:varchar(30)" json:"pauseReasonCode,omitempty"`
	PauseNote       *string      `json:"pauseNote,omitempty"`

	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	CreatedByID   string    `gorm:"type:uuid;not null" json:"createdById"`
	CreatedByName string    `gorm:"not null" json:"createdByName"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Milestone is a campaign's spend plan: what the money is for, in what
// order, and how much of it. The spec's example breaks a £100,000 goal into
// supplies/transport/distribution.
//
// It is also the withdrawal gate in Phase 5 — an org cannot draw funds
// without a milestone to draw them against. That is a deliberate departure
// from the spec listing "Withdraw funds" as a plain org capability:
// unilateral withdrawal would defeat the transparency the whole feature
// exists to provide.
type Milestone struct {
	ID          string          `gorm:"type:uuid;primaryKey" json:"id"`
	CampaignID  string          `gorm:"type:uuid;not null;index" json:"campaignId"`
	Title       string          `gorm:"not null" json:"title"`
	Description *string         `json:"description,omitempty"`
	TargetMinor int64           `gorm:"not null" json:"targetMinor"`
	Status      MilestoneStatus `gorm:"type:varchar(20);default:'PLANNED';index" json:"status"`
	// Position orders milestones in the plan. Same contract as
	// ConsultationQuestion.Position: the client supplies values, the server
	// sorts by it regardless of gaps.
	Position    int        `gorm:"not null" json:"position"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}
