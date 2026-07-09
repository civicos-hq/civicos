package domain

import "time"

type OrgKind string
type OrgJurisdiction string
type OrgMemberRole string
type AnnouncementStatus string
type ProjectStatus string
type AssignmentStatus string
type ConsultationStatus string
type QuestionType string

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
)

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
	CreatedByID       string          `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
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
	Options        []string     `gorm:"type:jsonb;serializer:json" json:"options"`
	Required       bool         `gorm:"default:false" json:"required"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
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
