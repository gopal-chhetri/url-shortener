package url

import (
	"context"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateURLDTO struct {
	OriginalURL string     `json:"original_url"`
	ShortURL    string     `json:"short_url"`
	UserID      *uuid.UUID `json:"user_id"`
	Tx          pgx.Tx     `json:"-"`
}

type GetURLByIDDTO struct {
	ID uuid.UUID `json:"id"`
	Tx pgx.Tx    `json:"-"`
}

type GetURLByShortURLDTO struct {
	ShortURL string `json:"short_url"`
	Tx       pgx.Tx `json:"-"`
}

type UpdateURLDTO struct {
	ID          uuid.UUID `json:"id"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	Tx          pgx.Tx    `json:"-"`
}

type DeleteURLDTO struct {
	ID uuid.UUID `json:"id"`
	Tx pgx.Tx    `json:"-"`
}

type ListURLsDTO struct {
	UserID uuid.UUID `json:"user_id"`
	Limit  int32     `json:"limit"`
	Offset int32     `json:"offset"`
	Tx     pgx.Tx    `json:"-"`
}

type GetURLCountDTO struct {
	UserID uuid.UUID `json:"user_id"`
	Tx     pgx.Tx    `json:"-"`
}

type ListAllURLsDTO struct {
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
	Tx     pgx.Tx `json:"-"`
}

type CountAllURLsDTO struct {
	Tx pgx.Tx `json:"-"`
}

type ListAllURLsByDateDTO struct {
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
	Tx     pgx.Tx `json:"-"`
}

type CountAllURLsByDateDTO struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Tx        pgx.Tx `json:"-"`
}

type UpdateURLStatusDTO struct {
	ID       uuid.UUID `json:"id"`
	IsActive bool      `json:"is_active"`
	Tx       pgx.Tx    `json:"-"`
}

type UrlRepositoryInterface interface {
	CreateURL(ctx context.Context, dto CreateURLDTO) (dbgen.Url, error)
	GetURLByID(ctx context.Context, dto GetURLByIDDTO) (dbgen.Url, error)
	GetURLByShortURL(ctx context.Context, dto GetURLByShortURLDTO) (dbgen.Url, error)
	UpdateURL(ctx context.Context, dto UpdateURLDTO) (dbgen.Url, error)
	UpdateURLStatus(ctx context.Context, dto UpdateURLStatusDTO) (dbgen.Url, error)
	DeleteURL(ctx context.Context, dto DeleteURLDTO) error
	ListURLs(ctx context.Context, dto ListURLsDTO) ([]dbgen.Url, error)
	GetURLCount(ctx context.Context, dto GetURLCountDTO) (int64, error)
	ListAllURLs(ctx context.Context, dto ListAllURLsDTO) ([]dbgen.Url, error)
	CountAllURLs(ctx context.Context, dto CountAllURLsDTO) (int64, error)
	ListAllURLsByDate(ctx context.Context, dto ListAllURLsByDateDTO) ([]dbgen.Url, error)
	CountAllURLsByDate(ctx context.Context, dto CountAllURLsByDateDTO) (int64, error)
	ListURLsByClicks(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbgen.Url, error)
	GetClickCountsByURLIDs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	GetPool() *pgxpool.Pool
}

type UrlRepository struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func NewUrlRepository(pool *pgxpool.Pool) UrlRepositoryInterface {
	return &UrlRepository{
		pool:    pool,
		queries: dbgen.New(pool),
	}
}

func (r *UrlRepository) GetPool() *pgxpool.Pool {
	return r.pool
}

func (r *UrlRepository) getQuerier(tx pgx.Tx) *dbgen.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func translateError(err error, model string) error {
	if err == nil {
		return nil
	}
	if err == pgx.ErrNoRows {
		return response.NotFoundError{Model: model}
	}
	return err
}

func (r *UrlRepository) CreateURL(ctx context.Context, dto CreateURLDTO) (dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)

	var userID pgtype.UUID
	if dto.UserID != nil {
		userID = pgtype.UUID{Bytes: *dto.UserID, Valid: true}
	}

	url, err := querier.CreateURL(ctx, dbgen.CreateURLParams{
		OriginalUrl: dto.OriginalURL,
		ShortUrl:    dto.ShortURL,
		UserID:      userID,
	})
	return url, translateError(err, "url")
}

func (r *UrlRepository) GetURLByID(ctx context.Context, dto GetURLByIDDTO) (dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	url, err := querier.GetURLByID(ctx, dto.ID)
	return url, translateError(err, "url")
}

func (r *UrlRepository) GetURLByShortURL(ctx context.Context, dto GetURLByShortURLDTO) (dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	url, err := querier.GetURLByShortURL(ctx, dto.ShortURL)
	return url, translateError(err, "url")
}

func (r *UrlRepository) UpdateURL(ctx context.Context, dto UpdateURLDTO) (dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	url, err := querier.UpdateURL(ctx, dbgen.UpdateURLParams{
		ID:          dto.ID,
		ShortUrl:    dto.ShortURL,
		OriginalUrl: dto.OriginalURL,
	})
	return url, translateError(err, "url")
}

func (r *UrlRepository) DeleteURL(ctx context.Context, dto DeleteURLDTO) error {
	querier := r.getQuerier(dto.Tx)
	return querier.DeleteURL(ctx, dto.ID)
}

func (r *UrlRepository) UpdateURLStatus(ctx context.Context, dto UpdateURLStatusDTO) (dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	url, err := querier.UpdateURLStatus(ctx, dbgen.UpdateURLStatusParams{
		ID:       dto.ID,
		IsActive: pgtype.Bool{Bool: dto.IsActive, Valid: true},
	})
	return url, translateError(err, "url")
}

func (r *UrlRepository) ListURLs(ctx context.Context, dto ListURLsDTO) ([]dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	urls, err := querier.ListURLs(ctx, dbgen.ListURLsParams{
		UserID: pgtype.UUID{Bytes: dto.UserID, Valid: true},
		Limit:  dto.Limit,
		Offset: dto.Offset,
	})
	return urls, translateError(err, "url")
}

func (r *UrlRepository) GetURLCount(ctx context.Context, dto GetURLCountDTO) (int64, error) {
	querier := r.getQuerier(dto.Tx)
	count, err := querier.GetURLCount(ctx, pgtype.UUID{Bytes: dto.UserID, Valid: true})
	return count, translateError(err, "url")
}

func (r *UrlRepository) ListAllURLs(ctx context.Context, dto ListAllURLsDTO) ([]dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	urls, err := querier.ListAllURLs(ctx, dbgen.ListAllURLsParams{
		Limit:  dto.Limit,
		Offset: dto.Offset,
	})
	return urls, translateError(err, "url")
}

func (r *UrlRepository) CountAllURLs(ctx context.Context, dto CountAllURLsDTO) (int64, error) {
	querier := r.getQuerier(dto.Tx)
	count, err := querier.CountAllURLs(ctx)
	return count, translateError(err, "url")
}

func (r *UrlRepository) ListAllURLsByDate(ctx context.Context, dto ListAllURLsByDateDTO) ([]dbgen.Url, error) {
	querier := r.getQuerier(dto.Tx)
	urls, err := querier.ListAllURLsByDate(ctx, dbgen.ListAllURLsByDateParams{
		Limit:  dto.Limit,
		Offset: dto.Offset,
	})
	return urls, translateError(err, "url")
}

func (r *UrlRepository) CountAllURLsByDate(ctx context.Context, dto CountAllURLsByDateDTO) (int64, error) {
	querier := r.getQuerier(dto.Tx)
	count, err := querier.CountAllURLsByDate(ctx, dbgen.CountAllURLsByDateParams{})
	return count, translateError(err, "url")
}

func (r *UrlRepository) ListURLsByClicks(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]dbgen.Url, error) {
	clickRows, err := r.queries.ListURLsByClicks(ctx, dbgen.ListURLsByClicksParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	urls := make([]dbgen.Url, len(clickRows))
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
	}
	return urls, nil
}

func (r *UrlRepository) GetClickCountsByURLIDs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	if len(urlIDs) == 0 {
		return map[uuid.UUID]int64{}, nil
	}

	rows, err := r.queries.GetClickCountsByURLIDs(ctx, urlIDs)
	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		counts[uuid.UUID(row.UrlID.Bytes)] = row.ClickCount
	}
	return counts, nil
}
