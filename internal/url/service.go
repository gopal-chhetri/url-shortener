package url

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type UrlServiceInterface interface {
	CreateURL(ctx context.Context, dto CreateURLRequest) (*dbgen.Url, error)
	GetURLByID(ctx context.Context, id uuid.UUID) (*dbgen.Url, error)
	GetURLByShortURL(ctx context.Context, shortURL string) (*dbgen.Url, error)
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

type CreateURLRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	CustomSlug  string     `json:"custom_slug,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
}

type UpdateURLRequest struct {
	ID          uuid.UUID `json:"id" binding:"required"`
	OriginalURL string    `json:"original_url" binding:"required,url"`
	CustomSlug  string    `json:"custom_slug,omitempty"`
}

type UrlService struct {
	repo      UrlRepositoryInterface
	clickRepo ClickRepositoryInterface
	redis     *redis.Client
	env       *infra.Env
	logger    *zap.Logger
}

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

// CreateURL creates a new shortened URL
func (s *UrlService) CreateURL(ctx context.Context, req CreateURLRequest) (*dbgen.Url, error) {
	// Validate original URL
	if req.OriginalURL == "" {
		return nil, response.NewAppError("Original URL is required")
	}

	// Generate or use custom slug
	var shortURL string
	if req.CustomSlug != "" {
		// Validate custom slug
		if !utils.IsBase62(req.CustomSlug) {
			return nil, response.NewAppError("Custom slug must contain only alphanumeric characters")
		}
		shortURL = req.CustomSlug
	} else {
		// Generate unique slug using timestamp
		shortURL = utils.GenerateUniqueSlug(req.OriginalURL, time.Now().UnixNano(), 7)
	}

	// Check if slug already exists
	_, err := s.repo.GetURLByShortURL(ctx, GetURLByShortURLDTO{ShortURL: shortURL})
	if err == nil {
		// Slug already exists, generate a new one with additional uniqueness
		s.logger.Warn("Slug collision detected, regenerating", zap.String("slug", shortURL))
		shortURL = utils.GenerateUniqueSlug(req.OriginalURL+uuid.New().String(), time.Now().UnixNano(), 7)
	}

	// Create URL in database
	url, err := s.repo.CreateURL(ctx, CreateURLDTO{
		OriginalURL: req.OriginalURL,
		ShortURL:    shortURL,
		UserID:      req.UserID,
	})
	if err != nil {
		s.logger.Error("Failed to create URL", zap.Error(err))
		return nil, err
	}

	s.logger.Info("URL created successfully",
		zap.String("short_url", shortURL),
		zap.String("original_url", req.OriginalURL),
		zap.Any("user_id", req.UserID),
	)

	return &url, nil
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
	key := "url:code:" + shortURL
	if s.redis != nil {
		val, err := s.redis.Get(ctx, key).Result()
		if err == nil {
			var cachedURL dbgen.Url
			if err := json.Unmarshal([]byte(val), &cachedURL); err == nil {
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

// UpdateURL updates a URL's details
func (s *UrlService) UpdateURL(ctx context.Context, req UpdateURLRequest, userID uuid.UUID) (*dbgen.Url, error) {
	// First check if URL exists and belongs to user
	existingURL, err := s.repo.GetURLByID(ctx, GetURLByIDDTO{ID: req.ID})
	if err != nil {
		s.logger.Warn("URL not found for update", zap.String("id", req.ID.String()))
		return nil, err
	}

	// Check ownership
	if existingURL.UserID.Valid && existingURL.UserID.Bytes != userID {
		s.logger.Warn("Unauthorized URL update attempt",
			zap.String("url_id", req.ID.String()),
			zap.String("user_id", userID.String()),
		)
		return nil, response.NewAppError("Unauthorized to update this URL")
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
		s.redis.Del(ctx, "url:code:"+existingURL.ShortUrl)
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

	// Check ownership
	if existingURL.UserID.Valid && existingURL.UserID.Bytes != userID {
		s.logger.Warn("Unauthorized URL deletion attempt",
			zap.String("url_id", id.String()),
			zap.String("user_id", userID.String()),
		)
		return response.NewAppError("Unauthorized to delete this URL")
	}

	// Delete URL
	if err := s.repo.DeleteURL(ctx, DeleteURLDTO{ID: id}); err != nil {
		s.logger.Error("Failed to delete URL", zap.Error(err))
		return err
	}

	// Invalidate cache
	if s.redis != nil {
		s.redis.Del(ctx, "url:code:"+existingURL.ShortUrl)
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
		allURLs, err := s.repo.ListURLs(ctx, ListURLsDTO{
			UserID: userID,
			Limit:  1000,
			Offset: 0,
		})
		if err != nil {
			s.logger.Error("Failed to list URLs", zap.Error(err))
			return nil, 0, err
		}

		urlIDs := make([]uuid.UUID, len(allURLs))
		for i, u := range allURLs {
			urlIDs[i] = u.ID
		}

		clickCounts, err := s.repo.GetClickCountsByURLIDs(ctx, urlIDs)
		if err != nil {
			s.logger.Error("Failed to get click counts", zap.Error(err))
			clickCounts = make(map[uuid.UUID]int64)
		}

		sort.Slice(allURLs, func(i, j int) bool {
			return clickCounts[allURLs[i].ID] > clickCounts[allURLs[j].ID]
		})

		start := offset
		if start > int32(len(allURLs)) {
			start = int32(len(allURLs))
		}
		end := start + limit
		if end > int32(len(allURLs)) {
			end = int32(len(allURLs))
		}
		urls = allURLs[start:end]
	} else {
		urls, err = s.repo.ListURLs(ctx, ListURLsDTO{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			s.logger.Error("Failed to list URLs", zap.Error(err))
			return nil, 0, err
		}
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

	if existing.UserID.Valid && existing.UserID.Bytes != userID {
		return nil, response.NewAppError("Unauthorized to modify this URL")
	}

	updated, err := s.repo.UpdateURLStatus(ctx, UpdateURLStatusDTO{ID: id, IsActive: isActive})
	if err != nil {
		s.logger.Error("Failed to update URL status", zap.Error(err))
		return nil, err
	}

	// Invalidate cache
	if s.redis != nil {
		s.redis.Del(ctx, "url:code:"+existing.ShortUrl)
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
	if existing.UserID.Valid && existing.UserID.Bytes != userID {
		return nil, response.NewAppError("Unauthorized")
	}

	basicStats, err := s.clickRepo.GetClickStatsByURLID(ctx, urlID)
	if err != nil {
		return nil, err
	}

	end := time.Now()
	start := end.AddDate(0, 0, -7)
	dailyClicks, err := s.clickRepo.GetClicksByDateRange(ctx, urlID, start.Format(time.RFC3339), end.Format(time.RFC3339))
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
