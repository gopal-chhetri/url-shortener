package admin

import (
	"context"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"go.uber.org/zap"
)

type AdminServiceInterface interface {
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)
	ListAllUsers(ctx context.Context, limit, offset int32) ([]dbgen.User, int64, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, roleName string) (dbgen.User, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) (dbgen.User, error)
	ListRoles(ctx context.Context) ([]dbgen.Role, error)
}

type TopURL struct {
	ID          string `json:"id"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	ClickCount  int64  `json:"click_count"`
}

type DashboardStats struct {
	TotalUsers   int64    `json:"total_users"`
	ActiveURLs   int64    `json:"active_urls"`
	InactiveURLs int64    `json:"inactive_urls"`
	TotalClicks  int64    `json:"total_clicks"`
	TopURLs      []TopURL `json:"top_urls"`
}

type AdminService struct {
	repo   AdminRepositoryInterface
	logger *zap.Logger
}

func NewAdminService(repo AdminRepositoryInterface, logger *zap.Logger) AdminServiceInterface {
	return &AdminService{repo: repo, logger: logger}
}

func (s *AdminService) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	totalUsers, err := s.repo.CountAllUsers(ctx)
	if err != nil {
		s.logger.Error("Failed to count users", zap.Error(err))
		return nil, err
	}

	activeURLs, err := s.repo.CountActiveURLs(ctx)
	if err != nil {
		s.logger.Error("Failed to count active URLs", zap.Error(err))
		return nil, err
	}

	inactiveURLs, err := s.repo.CountInactiveURLs(ctx)
	if err != nil {
		s.logger.Error("Failed to count inactive URLs", zap.Error(err))
		return nil, err
	}

	totalClicks, err := s.repo.CountAllClicks(ctx)
	if err != nil {
		s.logger.Error("Failed to count clicks", zap.Error(err))
		return nil, err
	}

	topRows, err := s.repo.GetTopURLsByClicks(ctx, 5)
	if err != nil {
		s.logger.Error("Failed to get top URLs", zap.Error(err))
		return nil, err
	}

	topURLs := make([]TopURL, len(topRows))
	for i, row := range topRows {
		topURLs[i] = TopURL{
			ID:          row.ID.String(),
			ShortURL:    row.ShortUrl,
			OriginalURL: row.OriginalUrl,
			ClickCount:  row.ClickCount,
		}
	}

	return &DashboardStats{
		TotalUsers:   totalUsers,
		ActiveURLs:   activeURLs,
		InactiveURLs: inactiveURLs,
		TotalClicks:  totalClicks,
		TopURLs:      topURLs,
	}, nil
}

func (s *AdminService) ListAllUsers(ctx context.Context, limit, offset int32) ([]dbgen.User, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	users, err := s.repo.ListAllUsers(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list users", zap.Error(err))
		return nil, 0, err
	}

	count, err := s.repo.CountAllUsers(ctx)
	if err != nil {
		s.logger.Error("Failed to count users", zap.Error(err))
		return nil, 0, err
	}

	return users, count, nil
}

func (s *AdminService) ListRoles(ctx context.Context) ([]dbgen.Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *AdminService) UpdateUserRole(ctx context.Context, userID uuid.UUID, roleName string) (dbgen.User, error) {
	role, err := s.repo.GetRoleByName(ctx, roleName)
	if err != nil {
		return dbgen.User{}, response.NewAppError("Role not found: " + roleName)
	}

	user, err := s.repo.UpdateUserRole(ctx, userID, role.ID)
	if err != nil {
		s.logger.Error("Failed to update user role", zap.Error(err))
		return dbgen.User{}, err
	}

	s.logger.Info("User role updated", zap.String("user_id", userID.String()), zap.String("role", roleName))
	return user, nil
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) (dbgen.User, error) {
	user, err := s.repo.UpdateUserStatus(ctx, userID, isActive)
	if err != nil {
		s.logger.Error("Failed to update user status", zap.Error(err))
		return dbgen.User{}, err
	}

	s.logger.Info("User status updated",
		zap.String("user_id", userID.String()),
		zap.Bool("is_active", isActive),
	)
	return user, nil
}
