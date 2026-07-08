package middleware

import (
	"net/http"

	"github.com/civicos/community-service/pkg/response"
	"github.com/gin-gonic/gin"
)

func ActiveCommunityID(c *gin.Context) string {
	value, _ := c.Get("activeCommunityID")
	activeCommunityID, _ := value.(string)
	return activeCommunityID
}

func PrimaryCommunityID(c *gin.Context) string {
	value, _ := c.Get("primaryCommunityID")
	id, _ := value.(string)
	return id
}

func CommunityMemberships(c *gin.Context) []string {
	value, _ := c.Get("communityMemberships")
	ids, _ := value.([]string)
	return ids
}

// RequirePrimaryCommunityMatch gates creation actions (issues, petitions,
// rep-profile edits) to the caller's home community — the one they registered
// with or explicitly promoted via the change-primary endpoint. Signal
// authenticity depends on knowing who "represents" this constituency, so
// creation is deliberately narrower than participation.
func RequirePrimaryCommunityMatch(c *gin.Context, targetCommunityID string) bool {
	primary := PrimaryCommunityID(c)
	if primary == "" {
		response.Error(c, http.StatusForbidden, "PRIMARY_COMMUNITY_REQUIRED", "Set a primary community to perform this action")
		c.Abort()
		return false
	}
	if primary != targetCommunityID {
		response.Error(c, http.StatusForbidden, "PRIMARY_COMMUNITY_MISMATCH", "This action can only be performed in your primary community")
		c.Abort()
		return false
	}
	return true
}

// RequireMembershipInCommunity gates participation actions (petition signing,
// comments, upvotes, reports) to any community the caller has joined. Wider
// than primary so citizens can lend signal to communities they care about
// without abandoning their home constituency.
func RequireMembershipInCommunity(c *gin.Context, targetCommunityID string) bool {
	for _, id := range CommunityMemberships(c) {
		if id == targetCommunityID {
			return true
		}
	}
	response.Error(c, http.StatusForbidden, "COMMUNITY_MEMBERSHIP_REQUIRED", "Join this community to perform this action")
	c.Abort()
	return false
}
