package auth

import (
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
	rg.POST("/verify-email", h.verifyEmail)
	rg.POST("/resend-verification", authMiddleware, h.resendVerification)
	rg.GET("/me", authMiddleware, h.me)
	rg.PATCH("/me", authMiddleware, h.updateMe)
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
		if err.Error() == "EMAIL_ALREADY_IN_USE" {
			response.Error(c, http.StatusConflict, "EMAIL_ALREADY_IN_USE", "This email is already registered")
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
		response.Error(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Token is invalid or expired")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"tokens": tokens})
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
