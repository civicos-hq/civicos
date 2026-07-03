package domain

import "time"

type OrgKind string
type OrgJurisdiction string
type OrgMemberRole string
type AnnouncementStatus string
type ProjectStatus string
type AssignmentStatus string

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
	CreatedByID       string          `gorm:"not null" json:"createdById"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

type OrgMember struct {
	ID             string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string        `gorm:"not null;uniqueIndex:idx_org_user" json:"organizationId"`
	UserID         string        `gorm:"not null;uniqueIndex:idx_org_user" json:"userId"`
	UserName       string        `gorm:"not null" json:"userName"`
	UserRole       string        `gorm:"not null" json:"userRole"`
	Role           OrgMemberRole `gorm:"type:varchar(20);default:'STAFF'" json:"role"`
	JoinedAt       time.Time     `json:"joinedAt"`
}

type Announcement struct {
	ID             string             `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string             `gorm:"not null;index" json:"organizationId"`
	Title          string             `gorm:"not null" json:"title"`
	Body           string             `gorm:"not null" json:"body"`
	Status         AnnouncementStatus `gorm:"type:varchar(20);default:'DRAFT'" json:"status"`
	PublishedAt    *time.Time         `json:"publishedAt,omitempty"`
	AuthorID       string             `gorm:"not null" json:"authorId"`
	AuthorName     string             `gorm:"not null" json:"authorName"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

type Project struct {
	ID              string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID  string        `gorm:"not null;index" json:"organizationId"`
	Title           string        `gorm:"not null" json:"title"`
	Description     string        `gorm:"not null" json:"description"`
	Status          ProjectStatus `gorm:"type:varchar(20);default:'PLANNED'" json:"status"`
	StartDate       *time.Time    `json:"startDate,omitempty"`
	ExpectedEndDate *time.Time    `json:"expectedEndDate,omitempty"`
	BudgetKobo      *int64        `json:"budgetKobo,omitempty"`
	CommunityID     *string       `gorm:"index" json:"communityId,omitempty"`
	CreatedByID     string        `gorm:"not null" json:"createdById"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// IssueAssignment records that an org has taken responsibility for an
// externally-owned Issue (owned by community-service). The IssueID is a
// bare UUID reference — this service does not FK across the boundary.
type IssueAssignment struct {
	ID             string           `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID string           `gorm:"not null;uniqueIndex:idx_org_issue" json:"organizationId"`
	IssueID        string           `gorm:"not null;uniqueIndex:idx_org_issue" json:"issueId"`
	Status         AssignmentStatus `gorm:"type:varchar(20);default:'RECEIVED'" json:"status"`
	Note           *string          `json:"note,omitempty"`
	AssignedByID   string           `gorm:"not null" json:"assignedById"`
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
	OrganizationID string    `gorm:"not null;index" json:"organizationId"`
	IssueID        *string   `gorm:"index" json:"issueId,omitempty"`
	ProjectID      *string   `gorm:"index" json:"projectId,omitempty"`
	Body           string    `gorm:"not null" json:"body"`
	IsPublic       bool      `gorm:"default:true" json:"isPublic"`
	AuthorID       string    `gorm:"not null" json:"authorId"`
	AuthorName     string    `gorm:"not null" json:"authorName"`
	CreatedAt      time.Time `json:"createdAt"`
}
