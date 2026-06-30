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
}

func NewAdminHandler(service AdminServiceInterface, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{service: service, logger: logger}
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
