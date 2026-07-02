package admin

import (
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

type AdminHandler struct {
	service AdminServiceInterface
	logger  *zap.Logger
	env     *infra.Env
}

func NewAdminHandler(service AdminServiceInterface, logger *zap.Logger, env *infra.Env) *AdminHandler {
	return &AdminHandler{service: service, logger: logger, env: env}
}

// UserResponse is the public response representation of a user
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

type AdminURLResponse struct {
	ID            string `json:"id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
	UserID        string `json:"user_id,omitempty"`
	UserFirstName string `json:"user_first_name,omitempty"`
	UserLastName  string `json:"user_last_name,omitempty"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	ClickCount    int64  `json:"click_count,omitempty"`
}

func toUserResponse(u dbgen.User, roleName string) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      roleName,
		IsActive:  u.IsActive.Bool,
		CreatedAt: u.CreatedAt.Time.String(),
	}
}

func (h *AdminHandler) toAdminURLResponse(u dbgen.Url, clickCount int64, user *dbgen.User) AdminURLResponse {
	var userID string
	var firstName, lastName string
	if u.UserID.Valid {
		userID = uuid.UUID(u.UserID.Bytes).String()
		if user != nil {
			firstName = user.FirstName
			lastName = user.LastName
		}
	}
	return AdminURLResponse{
		ID:            u.ID.String(),
		ShortURL:      h.buildFullShortURL(u.ShortUrl),
		OriginalURL:   u.OriginalUrl,
		UserID:        userID,
		UserFirstName: firstName,
		UserLastName:  lastName,
		IsActive:      u.IsActive.Bool,
		CreatedAt:     u.CreatedAt.Time.String(),
		UpdatedAt:     u.UpdatedAt.Time.String(),
		ClickCount:    clickCount,
	}
}

func (h *AdminHandler) buildFullShortURL(code string) string {
	baseURL := h.env.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return baseURL + "/" + code
}

// GetStats godoc
// @Summary Get admin dashboard statistics
// @Description Returns aggregate stats: total users, active/inactive URLs, total clicks, top 5 URLs
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=DashboardStats}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/stats [get]
func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetDashboardStats(c.Request.Context())
	if err != nil {
		infra.LogError(h.logger, "Failed to get dashboard stats", err)
		response.ErrorResponse(c, err)
		return
	}
	response.SuccessResponse(c, stats)
}

// ListUsers godoc
// @Summary List all users (admin)
// @Description Get a paginated list of all users in the system
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} response.PaginatedResponse{data=[]UserResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
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

	users, count, err := h.service.ListAllUsers(c.Request.Context(), int32(limit), offset)
	if err != nil {
		infra.LogError(h.logger, "Failed to list users", err)
		response.ErrorResponse(c, err)
		return
	}

	// Build role ID -> name lookup
	roles, _ := h.service.ListRoles(c.Request.Context())
	roleMap := make(map[string]string, len(roles))
	for _, r := range roles {
		roleMap[r.ID.String()] = r.Name
	}

	userResponses := make([]UserResponse, len(users))
	for i, u := range users {
		roleName := roleMap[u.RoleID.String()]
		if roleName == "" {
			roleName = "user"
		}
		userResponses[i] = toUserResponse(u, roleName)
	}

	pagination := utils.NewPaginationResponse(count, page, limit)
	response.SuccessPaginatedResponse(c, userResponses, pagination)
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole godoc
// @Summary Update user role (admin)
// @Description Assign a new role to a user
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body UpdateRoleRequest true "Role update request"
// @Success 200 {object} response.Response{data=UserResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/users/{id}/role [put]
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrorResponse(c, err)
		return
	}

	user, err := h.service.UpdateUserRole(c.Request.Context(), userID, req.Role)
	if err != nil {
		infra.LogError(h.logger, "Failed to update user role", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, toUserResponse(user, req.Role))
}

type UpdateStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateUserStatus godoc
// @Summary Toggle user active status (admin)
// @Description Activate or deactivate a user account
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body UpdateStatusRequest true "Status update request"
// @Success 200 {object} response.Response{data=UserResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/users/{id}/status [put]
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrorResponse(c, err)
		return
	}

	user, err := h.service.UpdateUserStatus(c.Request.Context(), userID, req.IsActive)
	if err != nil {
		infra.LogError(h.logger, "Failed to update user status", err)
		response.ErrorResponse(c, err)
		return
	}

	// Re-fetch user role name for the response
	roleName := "user" // Default after status update, role doesn't change
	response.SuccessResponse(c, toUserResponse(user, roleName))
}

// GetRoles godoc
// @Summary List all roles (admin)
// @Description Returns all available user roles
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /admin/roles [get]
func (h *AdminHandler) GetRoles(c *gin.Context) {
	// Hardcoded roles; in production these would come from DB
	roles := []gin.H{
		{"name": "admin", "description": "Full system access"},
		{"name": "staff", "description": "Staff access"},
		{"name": "user", "description": "Standard user access"},
	}
	c.JSON(http.StatusOK, gin.H{
		"error": false,
		"data":  roles,
	})
}

// ListURLs godoc
// @Summary List all URLs (admin)
// @Description Get a paginated list of all URLs with optional search and sort
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search term for short_url or original_url"
// @Param sort query string false "Sort by: date, clicks" default(date)
// @Success 200 {object} response.PaginatedResponse
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /admin/urls [get]
func (h *AdminHandler) ListURLs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "date")

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

	var urls []dbgen.Url
	var count int64
	var err error
	var clickCounts map[uuid.UUID]int64

	if search != "" {
		urls, count, err = h.service.SearchURLs(c.Request.Context(), search, int32(limit), offset)
	} else if sortBy == "clicks" {
		clickRows, clickCount, clickErr := h.service.ListURLsByClicks(c.Request.Context(), int32(limit), offset)
		if clickErr != nil {
			infra.LogError(h.logger, "Failed to list URLs by clicks", clickErr)
			response.ErrorResponse(c, clickErr)
			return
		}
		urls = make([]dbgen.Url, len(clickRows))
		clickCounts = make(map[uuid.UUID]int64, len(clickRows))
		for i, row := range clickRows {
			urls[i] = dbgen.Url{
				ID:          row.ID,
				ShortUrl:    row.ShortUrl,
				OriginalUrl: row.OriginalUrl,
				UserID:      row.UserID,
				IsActive:    row.IsActive,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			}
			clickCounts[row.ID] = row.ClickCount
		}
		count = clickCount
	} else if sortBy == "name" {
		urls, count, err = h.service.ListAllURLsByName(c.Request.Context(), int32(limit), offset)
	} else {
		urls, count, err = h.service.ListAllURLs(c.Request.Context(), int32(limit), offset)
	}

	if err != nil {
		infra.LogError(h.logger, "Failed to list URLs", err)
		response.ErrorResponse(c, err)
		return
	}

	// Fetch click counts for all URLs when not already populated (date/name/search sorts)
	if clickCounts == nil && len(urls) > 0 {
		urlIDs := make([]uuid.UUID, len(urls))
		for i, u := range urls {
			urlIDs[i] = u.ID
		}
		clickCounts, err = h.service.GetClickCountsByURLIDs(c.Request.Context(), urlIDs)
		if err != nil {
			infra.LogError(h.logger, "Failed to fetch click counts", err)
			clickCounts = make(map[uuid.UUID]int64)
		}
	}

	// Fetch user information for all URLs
	userIDs := make([]uuid.UUID, 0, len(urls))
	for _, u := range urls {
		if u.UserID.Valid {
			userIDs = append(userIDs, uuid.UUID(u.UserID.Bytes))
		}
	}

	usersMap := make(map[uuid.UUID]*dbgen.User)
	if len(userIDs) > 0 {
		users, err := h.service.GetUsersByIDs(c.Request.Context(), userIDs)
		if err != nil {
			infra.LogError(h.logger, "Failed to fetch users for URLs", err)
			// Continue without user info - not a critical error
		} else {
			for i := range users {
				usersMap[users[i].ID] = &users[i]
			}
		}
	}

	// Convert to AdminURLResponse for consistent JSON output
	urlResponses := make([]AdminURLResponse, len(urls))
	for i, u := range urls {
		clickCount := int64(0)
		if clickCounts != nil {
			clickCount = clickCounts[u.ID]
		}
		var user *dbgen.User
		if u.UserID.Valid {
			user = usersMap[uuid.UUID(u.UserID.Bytes)]
		}
		urlResponses[i] = h.toAdminURLResponse(u, clickCount, user)
	}

	pagination := utils.NewPaginationResponse(count, page, limit)
	response.SuccessPaginatedResponse(c, urlResponses, pagination)
}

// UpdateURLStatus godoc
// @Summary Toggle URL active status (admin)
// @Description Activate or deactivate any URL in the system
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Param request body UpdateStatusRequest true "Status update request"
// @Success 200 {object} response.Response{data=AdminURLResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/urls/{id}/status [put]
func (h *AdminHandler) UpdateURLStatus(c *gin.Context) {
	idStr := c.Param("id")
	urlID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationErrorResponse(c, err)
		return
	}

	url, err := h.service.UpdateURLStatus(c.Request.Context(), urlID, req.IsActive)
	if err != nil {
		infra.LogError(h.logger, "Failed to update URL status", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, h.toAdminURLResponse(url, 0, nil))
}

// DeleteURL godoc
// @Summary Delete URL (admin)
// @Description Soft-delete any URL in the system
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/urls/{id} [delete]
func (h *AdminHandler) DeleteURL(c *gin.Context) {
	idStr := c.Param("id")
	urlID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestResponse(c)
		return
	}

	if err := h.service.DeleteURL(c.Request.Context(), urlID); err != nil {
		infra.LogError(h.logger, "Failed to delete URL", err)
		response.ErrorResponse(c, err)
		return
	}

	response.SuccessResponse(c, gin.H{"message": "URL deleted"})
}
