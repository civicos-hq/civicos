package domain

import "time"

type IssueStatus string
type IssueCategory string
type PetitionStatus string
type NotificationType string

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

	NotificationIssueUpdate            NotificationType = "ISSUE_UPDATE"
	NotificationPetitionUpdate         NotificationType = "PETITION_UPDATE"
	NotificationRepresentativeResponse NotificationType = "REPRESENTATIVE_RESPONSE"
	// A representative published an announcement to their constituents.
	// Distinct from REPRESENTATIVE_RESPONSE, which is them replying to
	// something a citizen raised — a follower should be able to tell the
	// difference between "they answered you" and "they announced something".
	NotificationRepresentativeAnnouncement NotificationType = "REPRESENTATIVE_ANNOUNCEMENT"
	NotificationCommunityUpdate            NotificationType = "COMMUNITY_UPDATE"
	NotificationConsultationUpdate         NotificationType = "CONSULTATION_UPDATE"
	NotificationAnnouncementUpdate         NotificationType = "ANNOUNCEMENT_UPDATE"
	NotificationSystem                     NotificationType = "SYSTEM"

	// ─── Community Funding (Phase 4) ───
	//
	// Separate types rather than one CAMPAIGN_UPDATE catch-all: a donor
	// deciding whether to open a notification is answering different
	// questions for "the goal was reached" and "the money was spent on
	// something". Collapsing them would make the tray unfilterable exactly
	// where trust matters most.
	NotificationCampaignApproved   NotificationType = "CAMPAIGN_APPROVED"
	NotificationDonationReceived   NotificationType = "DONATION_RECEIVED"
	NotificationMilestoneCompleted NotificationType = "MILESTONE_COMPLETED"
	NotificationCampaignUpdate     NotificationType = "CAMPAIGN_UPDATE"
	NotificationFundingGoalReached NotificationType = "FUNDING_GOAL_REACHED"
	NotificationCampaignCompleted  NotificationType = "CAMPAIGN_COMPLETED"
)

type Notification struct {
	ID        string           `gorm:"type:uuid;primaryKey" json:"id"`
	Type      NotificationType `gorm:"type:varchar(40);not null" json:"type"`
	Title     string           `gorm:"not null" json:"title"`
	Body      string           `gorm:"not null" json:"body"`
	Read      bool             `gorm:"default:false;index" json:"read"`
	LinkURL   *string          `json:"linkUrl,omitempty"`
	UserID    string           `gorm:"type:uuid;not null;index" json:"userId"`
	CreatedAt time.Time        `json:"createdAt"`
}

type Community struct {
	ID          string     `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	Slug        string     `gorm:"uniqueIndex;not null" json:"slug"`
	Description *string    `json:"description,omitempty"`
	State       string     `gorm:"not null" json:"state"`
	LGA         string     `gorm:"not null" json:"lga"`
	Country     string     `gorm:"default:'Nigeria'" json:"country"`
	LogoURL     *string    `json:"logoUrl,omitempty"`
	CreatedByID string     `gorm:"type:uuid;not null" json:"createdById"`
	Issues      []Issue    `gorm:"foreignKey:CommunityID" json:"-"`
	Petitions   []Petition `gorm:"foreignKey:CommunityID" json:"-"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Issue struct {
	ID           string         `gorm:"type:uuid;primaryKey" json:"id"`
	Title        string         `gorm:"not null" json:"title"`
	Description  string         `gorm:"not null" json:"description"`
	Category     IssueCategory  `gorm:"type:varchar(30);default:'OTHER'" json:"category"`
	Status       IssueStatus    `gorm:"type:varchar(30);default:'OPEN'" json:"status"`
	Location     *string        `json:"location,omitempty"`
	ImageURLs    []string       `gorm:"type:jsonb;serializer:json" json:"imageUrls"`
	UpvoteCount  int            `gorm:"default:0" json:"upvoteCount"`
	CommentCount int            `gorm:"default:0" json:"commentCount"`
	CommunityID  string         `gorm:"type:uuid;not null;index" json:"communityId"`
	ReportedByID string         `gorm:"type:uuid;not null" json:"reportedById"`
	Comments     []IssueComment `gorm:"foreignKey:IssueID" json:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type IssueComment struct {
	ID                 string `gorm:"type:uuid;primaryKey" json:"id"`
	Content            string `gorm:"not null" json:"content"`
	IssueID            string `gorm:"type:uuid;not null;index" json:"issueId"`
	AuthorID           string `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName         string `gorm:"not null" json:"authorName"`
	AuthorRole         string `gorm:"not null" json:"authorRole"`
	IsOfficialResponse bool   `gorm:"default:false" json:"isOfficialResponse"`
	// IsHidden is computed at query time from content_flags — never stored.
	// When true the repository has already replaced Content and AuthorName
	// with placeholders so the citizen surface sees "[Removed by moderator]"
	// instead of the raw content while the row remains in-place (preserves
	// conversation flow and preserves the audit trail).
	IsHidden  bool      `gorm:"-" json:"isHidden"`
	CreatedAt time.Time `json:"createdAt"`
}

// RepresentativeAnnouncement is a representative speaking to their
// constituents directly.
//
// Until this existed, an elected official's only voice on CivicOS was a
// comment on somebody else's issue or petition — they could answer, but never
// raise something themselves. The Experience Architecture lists "public
// announcement capabilities" among what representatives need; this is that.
//
// It deliberately mirrors the organization announcement lifecycle
// (DRAFT → PUBLISHED → ARCHIVED) rather than inventing a second shape for the
// same idea. A representative should be able to prepare a statement before it
// is visible, and take one down without deleting the record that it was made.
type RepresentativeAnnouncement struct {
	ID               string `gorm:"type:uuid;primaryKey" json:"id"`
	RepresentativeID string `gorm:"type:uuid;not null;index" json:"representativeId"`
	// CommunityID is copied from the representative at creation so the
	// announcement can be scoped and searched without a join, and so it stays
	// attached to the constituency it was made in even if the profile is
	// later corrected.
	CommunityID string             `gorm:"type:uuid;not null;index" json:"communityId"`
	Title       string             `gorm:"not null" json:"title"`
	Body        string             `gorm:"not null" json:"body"`
	Status      AnnouncementStatus `gorm:"type:varchar(20);default:'DRAFT';index" json:"status"`
	PublishedAt *time.Time         `json:"publishedAt,omitempty"`
	// AuthorID is the user who wrote it — the representative's own account.
	// Stored alongside RepresentativeID because a profile outlives an
	// account, and "who actually said this" must survive that.
	AuthorID   string `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName string `gorm:"not null" json:"authorName"`
	// IsHidden is set by the moderation queue, never persisted here — the
	// flag lives in identity-service. Same arrangement as comments.
	IsHidden  bool      `gorm:"-" json:"isHidden"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AnnouncementStatus is shared with the organization announcement lifecycle
// in organization-service. Same words, same meanings, deliberately.
type AnnouncementStatus string

const (
	AnnouncementDraft     AnnouncementStatus = "DRAFT"
	AnnouncementPublished AnnouncementStatus = "PUBLISHED"
	AnnouncementArchived  AnnouncementStatus = "ARCHIVED"
)

type RepresentativeComment struct {
	ID                 string    `gorm:"type:uuid;primaryKey" json:"id"`
	Content            string    `gorm:"not null" json:"content"`
	RepresentativeID   string    `gorm:"type:uuid;not null;index" json:"representativeId"`
	AuthorID           string    `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName         string    `gorm:"not null" json:"authorName"`
	AuthorRole         string    `gorm:"not null" json:"authorRole"`
	IsOfficialResponse bool      `gorm:"default:false" json:"isOfficialResponse"`
	IsHidden           bool      `gorm:"-" json:"isHidden"`
	CreatedAt          time.Time `json:"createdAt"`
}

type PetitionComment struct {
	ID                 string    `gorm:"type:uuid;primaryKey" json:"id"`
	Content            string    `gorm:"not null" json:"content"`
	PetitionID         string    `gorm:"type:uuid;not null;index" json:"petitionId"`
	AuthorID           string    `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName         string    `gorm:"not null" json:"authorName"`
	AuthorRole         string    `gorm:"not null" json:"authorRole"`
	IsOfficialResponse bool      `gorm:"default:false" json:"isOfficialResponse"`
	IsHidden           bool      `gorm:"-" json:"isHidden"`
	CreatedAt          time.Time `json:"createdAt"`
}

type Petition struct {
	ID             string         `gorm:"type:uuid;primaryKey" json:"id"`
	Title          string         `gorm:"not null" json:"title"`
	Description    string         `gorm:"not null" json:"description"`
	Goal           int            `gorm:"not null" json:"goal"`
	SignatureCount int            `gorm:"default:0" json:"signatureCount"`
	CommentCount   int            `gorm:"default:0" json:"commentCount"`
	Status         PetitionStatus `gorm:"type:varchar(30);default:'DRAFT'" json:"status"`
	Deadline       *time.Time     `json:"deadline,omitempty"`
	ImageURLs      []string       `gorm:"type:jsonb;serializer:json" json:"imageUrls"`
	CommunityID    string         `gorm:"type:uuid;not null;index" json:"communityId"`
	CreatedByID    string         `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// IssueUpvote records that a specific user upvoted a specific issue. The
// compound unique index is what actually enforces "one vote per account" —
// before this table existed the service just kept incrementing the counter.
type IssueUpvote struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	IssueID   string    `gorm:"type:uuid;not null;uniqueIndex:idx_issue_upvoter" json:"issueId"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_issue_upvoter" json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type PetitionSignature struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	PetitionID string    `gorm:"type:uuid;not null;uniqueIndex:idx_petition_user" json:"petitionId"`
	UserID     string    `gorm:"type:uuid;not null;uniqueIndex:idx_petition_user" json:"userId"`
	CreatedAt  time.Time `json:"createdAt"`
}

type RepresentativeFollower struct {
	ID               string    `gorm:"type:uuid;primaryKey" json:"id"`
	RepresentativeID string    `gorm:"type:uuid;not null;uniqueIndex:idx_rep_follower" json:"representativeId"`
	UserID           string    `gorm:"type:uuid;not null;uniqueIndex:idx_rep_follower" json:"userId"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Representative struct {
	ID           string  `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string  `gorm:"not null" json:"name"`
	Title        string  `gorm:"not null" json:"title"`
	Position     string  `gorm:"not null" json:"position"`
	Constituency string  `gorm:"not null" json:"constituency"`
	Party        *string `json:"party,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	AvatarURL    *string `json:"avatarUrl,omitempty"`
	Email        *string `json:"email,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	Website      *string `json:"website,omitempty"`
	CommunityID  string  `gorm:"type:uuid;not null;index" json:"communityId"`
	// UserID is the account this profile belongs to, set when the person's
	// application is approved. Nullable: an admin-created or seeded profile
	// is unclaimed, and nobody may publish as it until it is linked.
	//
	// Not the same as CreatedByID, which records who inserted the row.
	UserID        *string   `gorm:"type:uuid" json:"-"`
	ResponseRate  int       `gorm:"default:0" json:"responseRate"`
	FollowerCount int       `gorm:"default:0" json:"followerCount"`
	CommentCount  int       `gorm:"default:0" json:"commentCount"`
	CreatedByID   string    `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
