package domain

import "time"

type IssueStatus string
type IssueCategory string
type PetitionStatus string

const (
	IssueStatusOpen        IssueStatus = "OPEN"
	IssueStatusUnderReview IssueStatus = "UNDER_REVIEW"
	IssueStatusInProgress  IssueStatus = "IN_PROGRESS"
	IssueStatusResolved    IssueStatus = "RESOLVED"
	IssueStatusClosed      IssueStatus = "CLOSED"

	CategoryInfrastructure IssueCategory = "INFRASTRUCTURE"
	CategoryHealth         IssueCategory = "HEALTH"
	CategoryEducation      IssueCategory = "EDUCATION"
	CategorySecurity       IssueCategory = "SECURITY"
	CategoryEnvironment    IssueCategory = "ENVIRONMENT"
	CategoryUtilities      IssueCategory = "UTILITIES"
	CategoryTransport      IssueCategory = "TRANSPORT"
	CategoryOther          IssueCategory = "OTHER"

	PetitionDraft      PetitionStatus = "DRAFT"
	PetitionActive     PetitionStatus = "ACTIVE"
	PetitionClosed     PetitionStatus = "CLOSED"
	PetitionSuccessful PetitionStatus = "SUCCESSFUL"
)

type Community struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Description *string   `json:"description,omitempty"`
	State       string    `gorm:"not null" json:"state"`
	LGA         string    `gorm:"not null" json:"lga"`
	Country     string    `gorm:"default:'Nigeria'" json:"country"`
	LogoURL     *string   `json:"logoUrl,omitempty"`
	CreatedByID string    `gorm:"not null" json:"createdById"`
	Issues      []Issue   `gorm:"foreignKey:CommunityID" json:"-"`
	Petitions   []Petition `gorm:"foreignKey:CommunityID" json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Issue struct {
	ID           string        `gorm:"type:uuid;primaryKey" json:"id"`
	Title        string        `gorm:"not null" json:"title"`
	Description  string        `gorm:"not null" json:"description"`
	Category     IssueCategory `gorm:"type:varchar(30);default:'OTHER'" json:"category"`
	Status       IssueStatus   `gorm:"type:varchar(30);default:'OPEN'" json:"status"`
	Location     *string       `json:"location,omitempty"`
	ImageURLs    []string      `gorm:"type:text[];serializer:json" json:"imageUrls"`
	UpvoteCount  int           `gorm:"default:0" json:"upvoteCount"`
	CommunityID  string        `gorm:"not null;index" json:"communityId"`
	ReportedByID string        `gorm:"not null" json:"reportedById"`
	Comments     []IssueComment `gorm:"foreignKey:IssueID" json:"-"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

type IssueComment struct {
	ID                 string    `gorm:"type:uuid;primaryKey" json:"id"`
	Content            string    `gorm:"not null" json:"content"`
	IssueID            string    `gorm:"not null;index" json:"issueId"`
	AuthorID           string    `gorm:"not null" json:"authorId"`
	IsOfficialResponse bool      `gorm:"default:false" json:"isOfficialResponse"`
	CreatedAt          time.Time `json:"createdAt"`
}

type Petition struct {
	ID             string         `gorm:"type:uuid;primaryKey" json:"id"`
	Title          string         `gorm:"not null" json:"title"`
	Description    string         `gorm:"not null" json:"description"`
	Goal           int            `gorm:"not null" json:"goal"`
	SignatureCount  int            `gorm:"default:0" json:"signatureCount"`
	Status         PetitionStatus `gorm:"type:varchar(30);default:'DRAFT'" json:"status"`
	Deadline       *time.Time     `json:"deadline,omitempty"`
	CommunityID    string         `gorm:"not null;index" json:"communityId"`
	CreatedByID    string         `gorm:"not null" json:"createdById"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type PetitionSignature struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	PetitionID string    `gorm:"not null;uniqueIndex:idx_petition_user" json:"petitionId"`
	UserID     string    `gorm:"not null;uniqueIndex:idx_petition_user" json:"userId"`
	CreatedAt  time.Time `json:"createdAt"`
}
