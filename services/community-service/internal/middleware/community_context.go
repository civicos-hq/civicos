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

func RequireActiveCommunityMatch(c *gin.Context, targetCommunityID string) bool {
	activeCommunityID := ActiveCommunityID(c)
	if activeCommunityID == "" {
		response.Error(c, http.StatusForbidden, "ACTIVE_COMMUNITY_REQUIRED", "Join and switch to a community to perform this action")
		c.Abort()
		return false
	}
	if activeCommunityID != targetCommunityID {
		response.Error(c, http.StatusForbidden, "ACTIVE_COMMUNITY_MISMATCH", "Switch your active community to perform this action here")
		c.Abort()
		return false
	}
	return true
}
