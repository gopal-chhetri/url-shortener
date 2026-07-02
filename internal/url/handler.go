package url

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
	"go.uber.org/zap"
)

type UrlHandler struct {
	urlService UrlServiceInterface
	logger     *zap.Logger
	env        *infra.Env
}

func NewUrlHandler(urlService UrlServiceInterface, logger *zap.Logger, env *infra.Env) *UrlHandler {
	return &UrlHandler{
		urlService: urlService,
		logger:     logger,
		env:        env,
	}
}

type UrlHandlerInterface interface {
	CreateURL(c *gin.Context)
	GetURLByID(c *gin.Context)
	RedirectURL(c *gin.Context)
	ListURLs(c *gin.Context)
	DeleteURL(c *gin.Context)
	UpdateURL(c *gin.Context)
	PatchURLStatus(c *gin.Context)
	GetURLAnalytics(c *gin.Context)
}

// URLResponse represents the response for a single URL
// swagger:model
type URLResponse struct {
	ID          string `json:"id"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// URLListResponse represents the response for a list of URLs
// swagger:model
type URLListResponse struct {
	URLs       []URLResponse             `json:"urls"`
	Pagination *utils.PaginationResponse `json:"pagination"`
}

// CreateURLRequest represents the request body for creating a URL
// swagger:model
type CreateURLRequestBody struct {
	OriginalURL string `json:"original_url" binding:"required,url" example:"https://example.com/very-long-url"`
	CustomSlug  string `json:"custom_slug,omitempty" example:"my-custom-slug"`
}

// UpdateURLRequestBody represents the request body for updating a URL
// swagger:model
type UpdateURLRequestBody struct {
	OriginalURL string `json:"original_url" binding:"required,url" example:"https://example.com/new-url"`
}

// toURLResponse converts db URL model to response format
func toURLResponse(url dbgen.Url) URLResponse {
	var userID string
	if url.UserID.Valid {
		userID = uuid.UUID(url.UserID.Bytes).String()
	}

	return URLResponse{
		ID:          url.ID.String(),
		ShortURL:    url.ShortUrl,
		OriginalURL: url.OriginalUrl,
		UserID:      userID,
		IsActive:    url.IsActive.Bool,
		CreatedAt:   url.CreatedAt.Time.String(),
		UpdatedAt:   url.UpdatedAt.Time.String(),
	}
}

// CreateURL godoc
// @Summary Create a short URL
// @Description Create a new shortened URL from an original URL
// @Tags urls
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateURLRequestBody true "Create URL request"
// @Success 201 {object} response.Response{data=URLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /urls [post]
func (h *UrlHandler) CreateURL(c *gin.Context) {
	var req CreateURLRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		infra.LogError(h.logger, "Failed to bind create URL request", err)
		response.ValidationErrorResponse(c, err)
		return
	}

	// Get authenticated user ID
	userIDStr := c.GetString("user_id")
	var userID *uuid.UUID
	if userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			infra.LogError(h.logger, "Invalid user ID", err)
			response.BadRequestResponse(c)
			return
		}
		userID = &id
	}

	// Create URL
	url, err := h.urlService.CreateURL(c.Request.Context(), CreateURLRequest{
		OriginalURL: req.OriginalURL,
		CustomSlug:  req.CustomSlug,
		UserID:      userID,
	})
	if err != nil {
		infra.LogError(h.logger, "Failed to create URL", err)
		response.ErrorResponse(c, err)
		return
	}

	// Build full short URL
	shortURL := h.buildFullShortURL(url.ShortUrl)

	resp := toURLResponse(*url)
	resp.ShortURL = shortURL

	response.SuccessCreatedResponse(c, resp)
}

// GetURLByID godoc
// @Summary Get URL by ID
// @Description Get a specific URL by its ID
// @Tags urls
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} response.Response{data=URLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /urls/{id} [get]
func (h *UrlHandler) GetURLByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		infra.LogError(h.logger, "Invalid URL ID", err)
		response.BadRequestResponse(c)
		return
	}

	url, err := h.urlService.GetURLByID(c.Request.Context(), id)
	if err != nil {
		infra.LogError(h.logger, "Failed to get URL", err)
		response.ErrorResponse(c, err)
		return
	}

	// Check ownership
	userIDStr := c.GetString("user_id")
	if url.UserID.Valid && userIDStr != "" {
		userID, _ := uuid.Parse(userIDStr)
		if uuid.UUID(url.UserID.Bytes) != userID {
			response.UnauthorizedResponse(c, "Unauthorized to access this URL")
			return
		}
	}

	resp := toURLResponse(*url)
	resp.ShortURL = h.buildFullShortURL(url.ShortUrl)

	response.SuccessResponse(c, resp)
}

// RedirectURL godoc
// @Summary Redirect to original URL
// @Description Redirect short URL to the original URL (no authentication required)
// @Tags urls
// @Param code path string true "Short URL code"
// @Success 302 "Redirect to original URL"
// @Failure 404 {object} response.Response
// @Router /{code} [get]
func (h *UrlHandler) RedirectURL(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.BadRequestResponse(c)
		return
	}

	url, err := h.urlService.GetURLByShortURL(c.Request.Context(), code)
	if err != nil {
		h.logger.Warn("URL not found for redirect", zap.String("code", code))
		response.NotFoundResponse(c, "URL not found")
		return
	}

	// Track click asynchronously
	go func() {
		// Parse user agent
		userAgent := c.GetHeader("User-Agent")
		deviceInfo := utils.ParseUserAgent(userAgent)

		// Get user ID if authenticated (optional)
		var userID *uuid.UUID
		if userIDStr := c.GetString("user_id"); userIDStr != "" {
			if id, err := uuid.Parse(userIDStr); err == nil {
				userID = &id
			}
		}

		// Track click using context.WithoutCancel to prevent cancellation when the HTTP request finishes
		if err := h.urlService.TrackClick(context.WithoutCancel(c.Request.Context()), url.ID, deviceInfo.Device, deviceInfo.Browser, userID); err != nil {
			h.logger.Error("Failed to track click", zap.Error(err))
		}
	}()

	h.logger.Info("URL redirected",
		zap.String("short_code", code),
		zap.String("original_url", url.OriginalUrl),
		zap.String("user_agent", c.GetHeader("User-Agent")),
		zap.String("ip", c.ClientIP()),
	)

	// Redirect to original URL
	c.Redirect(http.StatusFound, url.OriginalUrl)
}

// ListURLs godoc
// @Summary List user's URLs
// @Description Get a paginated list of URLs created by the authenticated user
// @Tags urls
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {object} response.PaginatedResponse{data=[]URLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /urls [get]
func (h *UrlHandler) ListURLs(c *gin.Context) {
	// Get authenticated user ID
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		infra.LogError(h.logger, "Invalid user ID", err)
		response.BadRequestResponse(c)
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := int32((page - 1) * limit)

	// Get URLs
	urls, count, err := h.urlService.ListURLs(c.Request.Context(), userID, int32(limit), offset)
	if err != nil {
		infra.LogError(h.logger, "Failed to list URLs", err)
		response.ErrorResponse(c, err)
		return
	}

	// Convert to response format
	urlResponses := make([]URLResponse, len(urls))
	for i, url := range urls {
		urlResponses[i] = toURLResponse(url)
		urlResponses[i].ShortURL = h.buildFullShortURL(url.ShortUrl)
	}

	// Build pagination metadata
	pagination := utils.NewPaginationResponse(count, page, limit)

	response.SuccessPaginatedResponse(c, urlResponses, pagination)
}

// DeleteURL godoc
// @Summary Delete a URL
// @Description Soft delete a URL (deactivate it)
// @Tags urls
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /urls/{id} [delete]
func (h *UrlHandler) DeleteURL(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		infra.LogError(h.logger, "Invalid URL ID", err)
		response.BadRequestResponse(c)
		return
	}

	// Get authenticated user ID
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		infra.LogError(h.logger, "Invalid user ID", err)
		response.BadRequestResponse(c)
		return
	}

	// Delete URL
	if err := h.urlService.DeleteURL(c.Request.Context(), id, userID); err != nil {
		infra.LogError(h.logger, "Failed to delete URL", err)
		response.ErrorResponse(c, err)
		return
	}

	h.logger.Info("URL deleted",
		zap.String("url_id", id.String()),
		zap.String("user_id", userID.String()),
	)

	response.SuccessResponse(c, gin.H{"message": "URL deleted successfully"})
}

// UpdateURL godoc
// @Summary Update a URL
// @Description Update the original URL for an existing short URL
// @Tags urls
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Param request body UpdateURLRequestBody true "Update URL request"
// @Success 200 {object} response.Response{data=URLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /urls/{id} [put]
func (h *UrlHandler) UpdateURL(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		infra.LogError(h.logger, "Invalid URL ID", err)
		response.BadRequestResponse(c)
		return
	}

	var req UpdateURLRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		infra.LogError(h.logger, "Failed to bind update URL request", err)
		response.ValidationErrorResponse(c, err)
		return
	}

	// Get authenticated user ID
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		infra.LogError(h.logger, "Invalid user ID", err)
		response.BadRequestResponse(c)
		return
	}

	// Update URL
	url, err := h.urlService.UpdateURL(c.Request.Context(), UpdateURLRequest{
		ID:          id,
		OriginalURL: req.OriginalURL,
	}, userID)
	if err != nil {
		infra.LogError(h.logger, "Failed to update URL", err)
		response.ErrorResponse(c, err)
		return
	}

	h.logger.Info("URL updated",
		zap.String("url_id", id.String()),
		zap.String("user_id", userID.String()),
	)

	resp := toURLResponse(*url)
	resp.ShortURL = h.buildFullShortURL(url.ShortUrl)

	response.SuccessResponse(c, resp)
}

// PatchURLStatus godoc
// @Summary Toggle URL active status
// @Description Activate or deactivate a URL (soft enable/disable)
// @Tags urls
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Param request body object true "Status update"
// @Success 200 {object} response.Response{data=URLResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /urls/{id}/status [patch]
func (h *UrlHandler) PatchURLStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrorResponse(c, err)
		return
	}

	url, err := h.urlService.UpdateURLStatus(c.Request.Context(), id, userID, req.IsActive)
	if err != nil {
		infra.LogError(h.logger, "Failed to update URL status", err)
		response.ErrorResponse(c, err)
		return
	}

	resp := toURLResponse(*url)
	resp.ShortURL = h.buildFullShortURL(url.ShortUrl)
	response.SuccessResponse(c, resp)
}

// GetURLAnalytics godoc
// @Summary Get URL analytics
// @Description Returns daily clicks (past 7 days), device breakdown, and browser breakdown
// @Tags urls
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /urls/{id}/analytics [get]
func (h *UrlHandler) GetURLAnalytics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.UnauthorizedResponse(c, "Unauthorized")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	analytics, err := h.urlService.GetURLAnalytics(c.Request.Context(), id, userID)
	if err != nil {
		infra.LogError(h.logger, "Failed to get URL analytics", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, analytics)
}

// buildFullShortURL builds the full short URL from the code
func (h *UrlHandler) buildFullShortURL(code string) string {
	// In production, this would use the base URL from environment
	baseURL := h.env.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return baseURL + "/" + code
}
