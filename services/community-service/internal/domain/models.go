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
	// NotificationFloodAlert carries a third-party flood forecast. Never
	// CivicOS's own prediction — the body attributes Google Flood Hub.
	NotificationFloodAlert NotificationType = "FLOOD_ALERT"

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
	ID          string  `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string  `gorm:"not null" json:"name"`
	Slug        string  `gorm:"uniqueIndex;not null" json:"slug"`
	Description *string `json:"description,omitempty"`
	State       string  `gorm:"not null" json:"state"`
	LGA         string  `gorm:"not null" json:"lga"`
	Country     string  `gorm:"default:'Nigeria'" json:"country"`
	// Latitude and Longitude locate the community as a point.
	//
	// Everything else here is an administrative NAME — "Lagos", "Ikeja" —
	// which is enough to group people but cannot answer "is this river
	// gauge near here?". Flood forecasts arrive as coordinates, so without
	// these a community cannot be matched to one.
	//
	// Nullable and set by an admin, never guessed. A community with no
	// coordinates is simply excluded from flood matching rather than
	// matched approximately: warning the wrong town is worse than warning
	// nobody, and a derived centroid would be a guess wearing the costume
	// of a measurement.
	//
	// Set together or not at all — see communities.Service.Update.
	Latitude    *float64   `gorm:"type:double precision" json:"latitude,omitempty"`
	Longitude   *float64   `gorm:"type:double precision" json:"longitude,omitempty"`
	LogoURL     *string    `json:"logoUrl,omitempty"`
	CreatedByID string     `gorm:"type:uuid;not null" json:"createdById"`
	Issues      []Issue    `gorm:"foreignKey:CommunityID" json:"-"`
	Petitions   []Petition `gorm:"foreignKey:CommunityID" json:"-"`
	// MemberCount is computed at query time from user_community_memberships
	// — a table owned by identity-service and read here from the shared
	// database, the same arrangement Discover uses for organizations and
	// campaigns. Never stored: `gorm:"-"` keeps it out of both AutoMigrate
	// and every SELECT, so the repository fills it explicitly.
	//
	// @civicos/types has promised this field since the Community interface
	// was written; until now nothing populated it, so every "N members"
	// label in the UI rendered "undefined members".
	MemberCount int       `gorm:"-" json:"memberCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	IsHidden     bool      `gorm:"-" json:"isHidden"`
	CommentCount int       `gorm:"default:0" json:"commentCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// RepresentativeAnnouncementComment is a constituent replying to a specific
// announcement.
//
// Attached to the announcement rather than to the representative, which the
// profile-level thread already covers. Without this, someone answering "Road
// works start Monday" would have to post into a general thread that may be
// about something else entirely — which makes an announcement a broadcast
// rather than a conversation, and broadcasting is what CivicOS exists to
// improve on.
type RepresentativeAnnouncementComment struct {
	ID             string `gorm:"type:uuid;primaryKey" json:"id"`
	AnnouncementID string `gorm:"type:uuid;not null;index" json:"announcementId"`
	Content        string `gorm:"not null" json:"content"`
	AuthorID       string `gorm:"type:uuid;not null" json:"authorId"`
	AuthorName     string `gorm:"not null" json:"authorName"`
	AuthorRole     string `gorm:"not null" json:"authorRole"`
	// IsOfficialResponse marks the representative answering on their own
	// announcement — the same badge citizens already read on issues and
	// petitions.
	IsOfficialResponse bool      `gorm:"default:false" json:"isOfficialResponse"`
	IsHidden           bool      `gorm:"-" json:"isHidden"`
	CreatedAt          time.Time `json:"createdAt"`
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

// CommunityFloodAlert is the flood forecast currently attached to a
// community, as issued by Google's Flood Forecasting API.
//
// Persisted rather than fetched on demand for three reasons: the upstream
// is rate-limited (200 requests/minute for the whole platform), a citizen
// opening the app during a flood must not depend on a third-party call
// succeeding, and notifications fire on CHANGE — which needs a previous
// value to compare against.
//
// One row per (community, gauge). A community near a confluence can be
// covered by several gauges, and collapsing them would hide the one that
// is rising.
type CommunityFloodAlert struct {
	ID          string `gorm:"type:uuid;primaryKey" json:"id"`
	CommunityID string `gorm:"type:uuid;not null;uniqueIndex:idx_community_gauge;index" json:"communityId"`
	GaugeID     string `gorm:"not null;uniqueIndex:idx_community_gauge" json:"gaugeId"`

	// Severity and Trend are Google's values, stored verbatim. Not
	// normalised into a CivicOS scale: a translation layer would make it
	// our judgement rather than theirs, which is exactly the attribution
	// this feature must not blur.
	Severity string `gorm:"type:varchar(20);not null;index" json:"severity"`
	Trend    string `gorm:"type:varchar(20)" json:"trend"`

	// River and SiteName come from the gauge metadata and exist to make an
	// alert legible. "The River Benue is forecast to flood" is actionable;
	// "gauge hybas_1121455890" is not.
	River    *string `json:"river,omitempty"`
	SiteName *string `json:"siteName,omitempty"`

	// DistanceKm is how far the gauge is from the community's point. Shown
	// to the citizen, because a warning from a gauge 40km upstream means
	// something different from one in town.
	DistanceKm float64 `json:"distanceKm"`

	GaugeLatitude  float64 `gorm:"type:double precision" json:"gaugeLatitude"`
	GaugeLongitude float64 `gorm:"type:double precision" json:"gaugeLongitude"`

	IssuedAt        time.Time  `json:"issuedAt"`
	ForecastStartAt *time.Time `json:"forecastStartAt,omitempty"`
	ForecastEndAt   *time.Time `json:"forecastEndAt,omitempty"`

	// NotifiedSeverity records what citizens were last told, which is not
	// the same as what the forecast last said. Without it an hourly poll
	// re-notifies an unchanged SEVERE every hour until people mute CivicOS
	// and miss the one that matters.
	NotifiedSeverity *string    `gorm:"type:varchar(20)" json:"-"`
	NotifiedAt       *time.Time `json:"-"`

	// LastSeenAt is when the upstream last reported this pairing. A row
	// that stops being reported is stale rather than safe — the UI hides
	// it, but it is kept so an operator can tell "the forecast cleared"
	// from "we stopped receiving forecasts".
	LastSeenAt time.Time `gorm:"index" json:"lastSeenAt"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
