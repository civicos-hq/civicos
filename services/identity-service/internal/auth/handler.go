package auth

import (
	"log"
	"net/http"

	"github.com/civicos/identity-service/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires the handler into the router.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	rg.POST("/register", h.register)
	rg.POST("/login", h.login)
	rg.POST("/refresh", h.refresh)
	rg.POST("/logout", h.logout)
	rg.POST("/verify-email", h.verifyEmail)
	rg.POST("/resend-verification", authMiddleware, h.resendVerification)
	rg.POST("/forgot-password", h.forgotPassword)
	rg.POST("/reset-password", h.resetPassword)
	rg.GET("/me", authMiddleware, h.me)
	rg.PATCH("/me", authMiddleware, h.updateMe)
	rg.DELETE("/me", authMiddleware, h.deleteMe)
	rg.POST("/me/community", authMiddleware, h.joinCommunity)
}

func (h *Handler) register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	user, tokens, err := h.service.Register(input)
	if err != nil {
		switch err.Error() {
		case "EMAIL_ALREADY_IN_USE":
			response.Error(c, http.StatusConflict, "EMAIL_ALREADY_IN_USE", "This email is already registered")
			return
		case "INVALID_REQUESTED_ACCOUNT_TYPE":
			response.Error(c, http.StatusBadRequest, "INVALID_REQUESTED_ACCOUNT_TYPE", "Choose citizen, representative, or organization")
			return
		case "REPRESENTATIVE_APPLICATION_REQUIRED":
			response.Error(c, http.StatusBadRequest, "REPRESENTATIVE_APPLICATION_REQUIRED", "Representative signup requires application details")
			return
		case "ORGANIZATION_APPLICATION_REQUIRED":
			response.Error(c, http.StatusBadRequest, "ORGANIZATION_APPLICATION_REQUIRED", "Organization signup requires application details")
			return
		case "STATE_REQUIRED":
			response.Error(c, http.StatusBadRequest, "STATE_REQUIRED", "This organization jurisdiction requires a state")
			return
		case "LGA_REQUIRED":
			response.Error(c, http.StatusBadRequest, "LGA_REQUIRED", "This organization jurisdiction requires an LGA")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Registration failed")
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"user": user, "tokens": tokens})
}

func (h *Handler) login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	user, tokens, err := h.service.Login(input)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"user": user, "tokens": tokens})
}

func (h *Handler) refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	tokens, err := h.service.RefreshTokens(body.RefreshToken)
	if err != nil {
		if err.Error() == "ACCOUNT_BANNED" {
			response.Error(c, http.StatusForbidden, "ACCOUNT_BANNED",
				"Your account has been suspended. Contact support if you believe this is a mistake.")
			return
		}
		if err.Error() == "ACCOUNT_DELETED" {
			response.Error(c, http.StatusForbidden, "ACCOUNT_DELETED",
				"This account has been deleted and cannot sign in.")
			return
		}
		response.Error(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Token is invalid or expired")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"tokens": tokens})
}

// deleteMe is the citizen-initiated self-service delete. Requires an
// authenticated Bearer token. Body accepts an optional { reason }. On
// success, response is 200 with an empty payload; the client is
// expected to clear its own local session state and navigate home.
func (h *Handler) deleteMe(c *gin.Context) {
	var body struct {
		Reason *string `json:"reason"`
	}
	// Body is optional — deleting without a reason is a valid choice.
	_ = c.ShouldBindJSON(&body)

	userID, _ := c.Get("userID")
	if err := h.service.DeleteAccount(userID.(string), body.Reason); err != nil {
		if err.Error() == "USER_NOT_FOUND" {
			response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete account")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) me(c *gin.Context) {
	userID, _ := c.Get("userID")
	user, err := h.service.GetMe(userID.(string))
	if err != nil {
		response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"user": user})
}

func (h *Handler) updateMe(c *gin.Context) {
	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	user, err := h.service.UpdateProfile(userID.(string), input)
	if err != nil {
		if err.Error() == "EMAIL_ALREADY_IN_USE" {
			response.Error(c, http.StatusConflict, "EMAIL_ALREADY_IN_USE", "This email is already registered")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update profile")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"user": user})
}

// logout accepts a refresh token in the body and revokes its family. Never
// gated by JWT auth: a user with an expired access token still needs to be
// able to sign out cleanly, and legitimate refresh material is proof enough.
// Unknown / already-revoked tokens still return 200 so the client can safely
// wipe local state either way.
func (h *Handler) logout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.service.Logout(body.RefreshToken); err != nil {
		log.Printf("[auth.logout] revoke failed (kept from client): %v", err)
	}
	response.Success(c, http.StatusOK, gin.H{"loggedOut": true})
}

func (h *Handler) verifyEmail(c *gin.Context) {
	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	user, tokens, err := h.service.VerifyEmail(body.Token)
	if err != nil {
		switch err.Error() {
		case "VERIFICATION_TOKEN_INVALID":
			response.Error(c, http.StatusBadRequest, "VERIFICATION_TOKEN_INVALID", "Invalid verification link")
		case "VERIFICATION_TOKEN_EXPIRED":
			response.Error(c, http.StatusBadRequest, "VERIFICATION_TOKEN_EXPIRED", "This verification link has expired. Request a new one.")
		default:
			response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not verify email")
		}
		return
	}
	payload := gin.H{"user": user}
	if tokens != nil {
		payload["tokens"] = tokens
	}
	response.Success(c, http.StatusOK, payload)
}

// forgotPassword always returns 200 — even for unknown emails — so an attacker
// cannot use response timing or status codes to enumerate accounts. Actual
// failures (SMTP down, DB error) log server-side and still respond 200.
func (h *Handler) forgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.service.RequestPasswordReset(body.Email); err != nil {
		// Log and swallow — do not leak whether the email exists / mailer failed.
		log.Printf("[auth.forgotPassword] backend error (kept from client): %v", err)
	}
	response.Success(c, http.StatusOK, gin.H{"sent": true})
}

func (h *Handler) resetPassword(c *gin.Context) {
	var body struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	user, tokens, err := h.service.ResetPassword(body.Token, body.NewPassword)
	if err != nil {
		switch err.Error() {
		case "RESET_TOKEN_INVALID":
			response.Error(c, http.StatusBadRequest, "RESET_TOKEN_INVALID", "Invalid reset link")
		case "RESET_TOKEN_EXPIRED":
			response.Error(c, http.StatusBadRequest, "RESET_TOKEN_EXPIRED", "This reset link has expired. Request a new one.")
		case "PASSWORD_TOO_SHORT":
			response.Error(c, http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters")
		default:
			response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not reset password")
		}
		return
	}
	response.Success(c, http.StatusOK, gin.H{"user": user, "tokens": tokens})
}

func (h *Handler) resendVerification(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.service.ResendVerification(userID.(string)); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not send verification email")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"sent": true})
}

func (h *Handler) joinCommunity(c *gin.Context) {
	var body struct {
		CommunityID string `json:"communityId" binding:"required,uuid4"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, _ := c.Get("userID")
	user, err := h.service.JoinCommunity(userID.(string), body.CommunityID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to join community")
		return
	}
	response.Success(c, http.StatusOK, gin.H{"user": user})
}
