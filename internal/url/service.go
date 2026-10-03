package url

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// UrlServiceInterface defines the behavior of the URL shortener business logic service.
// It supports URL creation, lookup, status updates, pagination, deletion, and analytics.
type UrlServiceInterface interface {
	CreateURL(ctx context.Context, dto CreateURLRequest) (*dbgen.Url, error)
	GetURLByID(ctx context.Context, id uuid.UUID) (*dbgen.Url, error)
	GetURLByShortURL(ctx context.Context, shortURL string) (*dbgen.Url, error)
	ExpireExpiredURLs(ctx context.Context) error
	UpdateURL(ctx context.Context, dto UpdateURLRequest, userID uuid.UUID) (*dbgen.Url, error)
	UpdateURLStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, isActive bool) (*dbgen.Url, error)
	DeleteURL(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ListURLs(ctx context.Context, userID uuid.UUID, limit, offset int32, sortBy string) ([]dbgen.Url, int64, error)
	GetURLCount(ctx context.Context, userID uuid.UUID) (int64, error)
	ListAllURLs(ctx context.Context, limit, offset int32) ([]dbgen.Url, int64, error)
	TrackClick(ctx context.Context, urlID uuid.UUID, device, browser string, userID *uuid.UUID, ipAddress, country, city string) error
	GetClickStats(ctx context.Context, urlID uuid.UUID) (dbgen.GetClickStatsByURLIDRow, error)
	GetURLAnalytics(ctx context.Context, urlID uuid.UUID, userID uuid.UUID) (*URLAnalytics, error)
	GetClickCountsByURLIDs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID]int64, error)
}

// CreateURLRequest contains parameters for shortening a new URL.
type CreateURLRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	CustomSlug  string     `json:"custom_slug,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	ExpiresAt   *time.Time `json:"-"`
}

// UpdateURLRequest contains details for updating the long URL or custom slug of an existing code.
type UpdateURLRequest struct {
	ID          uuid.UUID `json:"id" binding:"required"`
	OriginalURL string    `json:"original_url" binding:"required,url"`
	CustomSlug  string    `json:"custom_slug,omitempty"`
}

// UrlService provides implementation for UrlServiceInterface, managing URL lifecycle and analytics.
type UrlService struct {
	repo      UrlRepositoryInterface
	clickRepo ClickRepositoryInterface
	redis     *redis.Client
	env       *infra.Env
	logger    *zap.Logger
}

// NewUrlService initializes a basic UrlService instance with database, redis cache, environment, and logger.
func NewUrlService(repo UrlRepositoryInterface, redis *redis.Client, env *infra.Env, logger *zap.Logger) UrlServiceInterface {
	return &UrlService{
		repo:   repo,
		redis:  redis,
		env:    env,
		logger: logger,
	}
}

// NewUrlServiceWithClicks creates a URL service with click tracking
func NewUrlServiceWithClicks(repo UrlRepositoryInterface, clickRepo ClickRepositoryInterface, redis *redis.Client, env *infra.Env, logger *zap.Logger) UrlServiceInterface {
	return &UrlService{
		repo:      repo,
		clickRepo: clickRepo,
		redis:     redis,
		env:       env,
		logger:    logger,
	}
}

// maxSlugAttempts bounds how many generated slugs CreateURL tries before
// giving up on finding an unused one.
const maxSlugAttempts = 5

// CreateURL creates a new shortened URL
func (s *UrlService) CreateURL(ctx context.Context, req CreateURLRequest) (*dbgen.Url, error) {
	if !utils.IsHTTPURL(req.OriginalURL) {
		return nil, response.NewAppError("Original URL must be an http or https URL")
	}

	dto := CreateURLDTO{
		OriginalURL: req.OriginalURL,
		UserID:      req.UserID,
		ExpiresAt:   req.ExpiresAt,
	}

	// A requested custom slug is used as-is; if it's taken the caller gets a
	// 409 rather than a silently different link.
	if req.CustomSlug != "" {
		if !utils.IsValidSlug(req.CustomSlug) {
			return nil, response.NewAppError(fmt.Sprintf("Custom slug must be at most %d letters, numbers, hyphens, or underscores, and not a reserved word", utils.MaxSlugLength))
		}
		dto.ShortURL = req.CustomSlug
		url, err := s.repo.CreateURL(ctx, dto)
		if err != nil {
			var dup response.DuplicateData
			if errors.As(err, &dup) {
				return nil, response.DuplicateData{Model: "custom slug"}
			}
			s.logger.Error("Failed to create URL", zap.Error(err))
			return nil, err
		}
		s.logCreated(url)
		return &url, nil
	}

	// Generated slugs rely on the unique constraint instead of a racy
	// pre-check: on collision, salt the input and try again.
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		seed := req.OriginalURL
		if attempt > 0 {
			seed += uuid.New().String()
		}
		dto.ShortURL = utils.GenerateUniqueSlug(seed, time.Now().UnixNano(), 7)

		url, err := s.repo.CreateURL(ctx, dto)
		if err == nil {
			s.logCreated(url)
			return &url, nil
		}
		var dup response.DuplicateData
		if !errors.As(err, &dup) {
			s.logger.Error("Failed to create URL", zap.Error(err))
			return nil, err
		}
		s.logger.Warn("Slug collision detected, regenerating", zap.String("slug", dto.ShortURL))
	}
	return nil, fmt.Errorf("could not generate a unique slug after %d attempts", maxSlugAttempts)
}

// logCreated records a successfully created URL.
func (s *UrlService) logCreated(url dbgen.Url) {
	s.logger.Info("URL created successfully",
		zap.String("short_url", url.ShortUrl),
		zap.String("original_url", url.OriginalUrl),
		zap.Bool("anonymous", !url.UserID.Valid),
	)
}

// requireOwner reports a not-found error unless userID owns url. Anonymous
// (ownerless) URLs belong to nobody, so no account can modify them, and
// non-owners can't distinguish someone else's URL from a missing one.
func requireOwner(url dbgen.Url, userID uuid.UUID) error {
	if !url.UserID.Valid || uuid.UUID(url.UserID.Bytes) != userID {
		return response.NotFoundError{Model: "url"}
	}
	return nil
}

// GetURLByID retrieves a URL by its ID
func (s *UrlService) GetURLByID(ctx context.Context, id uuid.UUID) (*dbgen.Url, error) {
	url, err := s.repo.GetURLByID(ctx, GetURLByIDDTO{ID: id})
	if err != nil {
		s.logger.Warn("URL not found", zap.String("id", id.String()))
		return nil, err
	}

	return &url, nil
}

// GetURLByShortURL retrieves a URL by its short code (for redirection)
func (s *UrlService) GetURLByShortURL(ctx context.Context, shortURL string) (*dbgen.Url, error) {
	key := infra.URLCacheKey(shortURL)
	if s.redis != nil {
		val, err := s.redis.Get(ctx, key).Result()
		if err == nil {
			var cachedURL dbgen.Url
			if err := json.Unmarshal([]byte(val), &cachedURL); err == nil {
				if cachedURL.ExpiresAt.Valid && cachedURL.ExpiresAt.Time.Before(time.Now()) {
					s.redis.Del(ctx, key)
					return nil, response.NotFoundError{Model: "url"}
				}
				return &cachedURL, nil
			}
		}
	}

	url, err := s.repo.GetURLByShortURL(ctx, GetURLByShortURLDTO{ShortURL: shortURL})
	if err != nil {
		s.logger.Warn("URL not found", zap.String("short_url", shortURL))
		return nil, err
	}

	if s.redis != nil {
		if jsonData, err := json.Marshal(url); err == nil {
			s.redis.Set(ctx, key, jsonData, 10*time.Minute)
		}
	}

	return &url, nil
}

// ExpireExpiredURLs marks expired anonymous URLs as inactive.
func (s *UrlService) ExpireExpiredURLs(ctx context.Context) error {
	return s.repo.ExpireExpiredURLs(ctx)
}

// UpdateURL updates a URL's details
func (s *UrlService) UpdateURL(ctx context.Context, req UpdateURLRequest, userID uuid.UUID) (*dbgen.Url, error) {
	if !utils.IsHTTPURL(req.OriginalURL) {
		return nil, response.NewAppError("Original URL must be an http or https URL")
	}

	// First check if URL exists and belongs to user
	existingURL, err := s.repo.GetURLByID(ctx, GetURLByIDDTO{ID: req.ID})
	if err != nil {
		s.logger.Warn("URL not found for update", zap.String("id", req.ID.String()))
		return nil, err
	}

	if err := requireOwner(existingURL, userID); err != nil {
		s.logger.Warn("Unauthorized URL update attempt",
			zap.String("url_id", req.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, err
	}

	// Update URL
	updatedURL, err := s.repo.UpdateURL(ctx, UpdateURLDTO{
		ID:          req.ID,
		OriginalURL: req.OriginalURL,
		ShortURL:    existingURL.ShortUrl, // Keep existing short URL
	})
	if err != nil {
		s.logger.Error("Failed to update URL", zap.Error(err))
		return nil, err
	}

	// Invalidate cache
	if s.redis != nil {
		s.redis.Del(ctx, infra.URLCacheKey(existingURL.ShortUrl))
	}

	s.logger.Info("URL updated successfully",
		zap.String("id", req.ID.String()),
		zap.String("user_id", userID.String()),
	)

	return &updatedURL, nil
}

// DeleteURL soft deletes a URL (sets is_active to false)
func (s *UrlService) DeleteURL(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// First check if URL exists and belongs to user
	existingURL, err := s.repo.GetURLByID(ctx, GetURLByIDDTO{ID: id})
	if err != nil {
		s.logger.Warn("URL not found for deletion", zap.String("id", id.String()))
		return err
	}

	if err := requireOwner(existingURL, userID); err != nil {
		s.logger.Warn("Unauthorized URL deletion attempt",
			zap.String("url_id", id.String()),
			zap.String("user_id", userID.String()),
		)
		return err
	}

	// Delete URL
	if err := s.repo.DeleteURL(ctx, DeleteURLDTO{ID: id}); err != nil {
		s.logger.Error("Failed to delete URL", zap.Error(err))
		return err
	}

	// Invalidate cache
	if s.redis != nil {
		s.redis.Del(ctx, infra.URLCacheKey(existingURL.ShortUrl))
	}

	s.logger.Info("URL deleted successfully",
		zap.String("id", id.String()),
		zap.String("user_id", userID.String()),
	)

	return nil
}

func (s *UrlService) ListURLs(ctx context.Context, userID uuid.UUID, limit, offset int32, sortBy string) ([]dbgen.Url, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var urls []dbgen.Url
	var err error

	if sortBy == "clicks" {
		urls, err = s.repo.ListURLsByClicks(ctx, userID, limit, offset)
	} else {
		urls, err = s.repo.ListURLs(ctx, ListURLsDTO{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		s.logger.Error("Failed to list URLs", zap.Error(err))
		return nil, 0, err
	}

	count, err := s.repo.GetURLCount(ctx, GetURLCountDTO{UserID: userID})
	if err != nil {
		s.logger.Error("Failed to get URL count", zap.Error(err))
		return nil, 0, err
	}

	return urls, count, nil
}

func (s *UrlService) GetClickCountsByURLIDs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return s.repo.GetClickCountsByURLIDs(ctx, urlIDs)
}

// GetURLCount returns the total count of URLs for a user
func (s *UrlService) GetURLCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.repo.GetURLCount(ctx, GetURLCountDTO{UserID: userID})
	if err != nil {
		s.logger.Error("Failed to get URL count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// ListAllURLs retrieves all URLs (admin only)
func (s *UrlService) ListAllURLs(ctx context.Context, limit, offset int32) ([]dbgen.Url, int64, error) {
	// Set default limit if not provided
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Get all URLs
	urls, err := s.repo.ListAllURLs(ctx, ListAllURLsDTO{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		s.logger.Error("Failed to list all URLs", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	count, err := s.repo.CountAllURLs(ctx, CountAllURLsDTO{})
	if err != nil {
		s.logger.Error("Failed to count all URLs", zap.Error(err))
		return nil, 0, err
	}

	return urls, count, nil
}

// TrackClick records a click event for a URL
func (s *UrlService) TrackClick(ctx context.Context, urlID uuid.UUID, device, browser string, userID *uuid.UUID, ipAddress, country, city string) error {
	if s.clickRepo == nil {
		s.logger.Warn("Click tracking disabled - click repository not initialized")
		return nil
	}

	_, err := s.clickRepo.CreateClick(ctx, CreateClickDTO{
		UrlID:     urlID,
		UserID:    userID,
		Device:    device,
		Browser:   browser,
		IPAddress: ipAddress,
		Country:   country,
		City:      city,
	})
	if err != nil {
		s.logger.Error("Failed to track click", zap.Error(err), zap.String("url_id", urlID.String()))
		return err
	}

	s.logger.Info("Click tracked",
		zap.String("url_id", urlID.String()),
		zap.String("device", device),
		zap.String("browser", browser),
		zap.String("ip", ipAddress),
		zap.String("country", country),
		zap.String("city", city),
	)

	return nil
}

// GetClickStats returns click statistics for a URL
func (s *UrlService) GetClickStats(ctx context.Context, urlID uuid.UUID) (dbgen.GetClickStatsByURLIDRow, error) {
	if s.clickRepo == nil {
		return dbgen.GetClickStatsByURLIDRow{}, response.NewAppError("Click tracking not enabled")
	}

	stats, err := s.clickRepo.GetClickStatsByURLID(ctx, urlID)
	if err != nil {
		s.logger.Error("Failed to get click stats", zap.Error(err))
		return dbgen.GetClickStatsByURLIDRow{}, err
	}

	return stats, nil
}

// URLAnalytics holds per-URL analytics data
type URLAnalytics struct {
	TotalClicks  int64                               `json:"total_clicks"`
	DailyClicks  []dbgen.GetClickStatsByDateRangeRow `json:"daily_clicks"`
	DeviceStats  []dbgen.GetDeviceStatsByURLIDRow    `json:"device_stats"`
	BrowserStats []dbgen.GetBrowserStatsByURLIDRow   `json:"browser_stats"`
	GeoStats     []dbgen.GetGeoStatsByURLIDRow       `json:"geo_stats"`
}

// UpdateURLStatus toggles the active status of a URL
func (s *UrlService) UpdateURLStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, isActive bool) (*dbgen.Url, error) {
	existing, err := s.repo.GetURLByID(ctx, GetURLByIDDTO{ID: id})
	if err != nil {
		return nil, err
	}

	if err := requireOwner(existing, userID); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateURLStatus(ctx, UpdateURLStatusDTO{ID: id, IsActive: isActive})
	if err != nil {
		s.logger.Error("Failed to update URL status", zap.Error(err))
		return nil, err
	}

	// Invalidate cache
	if s.redis != nil {
		s.redis.Del(ctx, infra.URLCacheKey(existing.ShortUrl))
	}

	return &updated, nil
}

// GetURLAnalytics returns analytics for a URL (daily clicks + device/browser breakdown)
func (s *UrlService) GetURLAnalytics(ctx context.Context, urlID uuid.UUID, userID uuid.UUID) (*URLAnalytics, error) {
	if s.clickRepo == nil {
		return nil, response.NewAppError("Click tracking not enabled")
	}

	// Verify ownership
	existing, err := s.repo.GetURLByID(ctx, GetURLByIDDTO{ID: urlID})
	if err != nil {
		return nil, err
	}
	if err := requireOwner(existing, userID); err != nil {
		return nil, err
	}

	basicStats, err := s.clickRepo.GetClickStatsByURLID(ctx, urlID)
	if err != nil {
		return nil, err
	}

	end := time.Now()
	start := end.AddDate(0, 0, -7)
	dailyClicks, err := s.clickRepo.GetClicksByDateRange(ctx, urlID, start, end)
	if err != nil {
		dailyClicks = nil
	}

	deviceStats, err := s.clickRepo.GetDeviceStatsByURLID(ctx, urlID)
	if err != nil {
		deviceStats = nil
	}

	browserStats, err := s.clickRepo.GetBrowserStatsByURLID(ctx, urlID)
	if err != nil {
		browserStats = nil
	}

	geoStats, err := s.clickRepo.GetGeoStatsByURLID(ctx, urlID)
	if err != nil {
		geoStats = nil
	}

	return &URLAnalytics{
		TotalClicks:  basicStats.TotalClicks,
		DailyClicks:  dailyClicks,
		DeviceStats:  deviceStats,
		BrowserStats: browserStats,
		GeoStats:     geoStats,
	}, nil
}
