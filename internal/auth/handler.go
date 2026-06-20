package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService AuthServiceInterface
	logger      *zap.Logger
}

func NewAuthHandler(authService AuthServiceInterface, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

type AuthHandlerInterface interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	RefreshToken(c *gin.Context)
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register request"
// @Success 201 {object} response.Response{data=AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		infra.LogError(h.logger, "Failed to bind register request", err)
		response.ValidationErrorResponse(c, err)
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		infra.LogError(h.logger, "Registration failed", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessCreatedResponse(c, resp)
}

// Login godoc
// @Summary Login user
// @Description Login user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} response.Response{data=AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		infra.LogError(h.logger, "Failed to bind login request", err)
		response.ValidationErrorResponse(c, err)
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		infra.LogError(h.logger, "Login failed", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, resp)
}

// Logout godoc
// @Summary Logout user
// @Description Logout user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		infra.LogError(h.logger, "Invalid user ID", err)
		response.UnauthorizedResponse(c, "Invalid token")
		return
	}

	err = h.authService.Logout(c.Request.Context(), userUUID)
	if err != nil {
		infra.LogError(h.logger, "Logout failed", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, nil)
}

// RefreshToken godoc
// @Summary Refresh JWT token
// @Description Refresh JWT token for authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=TokenResponse}
// @Failure 401 {object} response.Response
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		infra.LogError(h.logger, "Invalid user ID", err)
		response.BadRequestResponse(c)
		return
	}

	token, err := h.authService.RefreshToken(c.Request.Context(), userUUID)
	if err != nil {
		infra.LogError(h.logger, "Token refresh failed", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, TokenResponse{Token: token})
}
