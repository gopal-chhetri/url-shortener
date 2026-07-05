package url

import (
	"context"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateClickDTO struct {
	UserID    *uuid.UUID
	UrlID     uuid.UUID
	Device    string
	Browser   string
	IPAddress string
	Country   string
	City      string
	Latitude  float64
	Longitude float64
	Tx        pgx.Tx
}

type GetClicksByURLDTO struct {
	UrlID  uuid.UUID
	Limit  int32
	Offset int32
	Tx     pgx.Tx
}

type GetClicksByUserDTO struct {
	UserID uuid.UUID
	Limit  int32
	Offset int32
	Tx     pgx.Tx
}

type ClickRepositoryInterface interface {
	CreateClick(ctx context.Context, dto CreateClickDTO) (dbgen.Click, error)
	GetClicksByURLID(ctx context.Context, dto GetClicksByURLDTO) ([]dbgen.Click, error)
	GetClickCountByURLID(ctx context.Context, urlID uuid.UUID) (int64, error)
	GetClicksByUserID(ctx context.Context, dto GetClicksByUserDTO) ([]dbgen.Click, error)
	GetClickCountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetClickStatsByURLID(ctx context.Context, urlID uuid.UUID) (dbgen.GetClickStatsByURLIDRow, error)
	GetDeviceStatsByURLID(ctx context.Context, urlID uuid.UUID) ([]dbgen.GetDeviceStatsByURLIDRow, error)
	GetBrowserStatsByURLID(ctx context.Context, urlID uuid.UUID) ([]dbgen.GetBrowserStatsByURLIDRow, error)
	GetGeoStatsByURLID(ctx context.Context, urlID uuid.UUID) ([]dbgen.GetGeoStatsByURLIDRow, error)
	GetClicksByDateRange(ctx context.Context, urlID uuid.UUID, start, end string) ([]dbgen.GetClickStatsByDateRangeRow, error)
}

type ClickRepository struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func NewClickRepository(pool *pgxpool.Pool) ClickRepositoryInterface {
	return &ClickRepository{
		pool:    pool,
		queries: dbgen.New(pool),
	}
}

func (r *ClickRepository) getQuerier(tx pgx.Tx) *dbgen.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *ClickRepository) CreateClick(ctx context.Context, dto CreateClickDTO) (dbgen.Click, error) {
	querier := r.getQuerier(dto.Tx)

	var userID pgtype.UUID
	if dto.UserID != nil {
		userID = pgtype.UUID{Bytes: *dto.UserID, Valid: true}
	}

	var latitude, longitude pgtype.Numeric
	latitude.Valid = false
	longitude.Valid = false

	var ipAddress, country, city pgtype.Text
	if dto.IPAddress != "" {
		ipAddress = pgtype.Text{String: dto.IPAddress, Valid: true}
	}
	if dto.Country != "" {
		country = pgtype.Text{String: dto.Country, Valid: true}
	}
	if dto.City != "" {
		city = pgtype.Text{String: dto.City, Valid: true}
	}

	var device, browser pgtype.Text
	if dto.Device != "" {
		device = pgtype.Text{String: dto.Device, Valid: true}
	}
	if dto.Browser != "" {
		browser = pgtype.Text{String: dto.Browser, Valid: true}
	}

	click, err := querier.CreateClick(ctx, dbgen.CreateClickParams{
		UserID:    userID,
		UrlID:     pgtype.UUID{Bytes: dto.UrlID, Valid: true},
		Device:    device,
		Browser:   browser,
		IpAddress: ipAddress,
		Country:   country,
		City:      city,
		Latitude:  latitude,
		Longitude: longitude,
	})

	return click, translateError(err, "click")
}

func (r *ClickRepository) GetClicksByURLID(ctx context.Context, dto GetClicksByURLDTO) ([]dbgen.Click, error) {
	querier := r.getQuerier(dto.Tx)
	clicks, err := querier.GetClicksByURLID(ctx, dbgen.GetClicksByURLIDParams{
		UrlID:  pgtype.UUID{Bytes: dto.UrlID, Valid: true},
		Limit:  dto.Limit,
		Offset: dto.Offset,
	})
	return clicks, translateError(err, "click")
}

func (r *ClickRepository) GetClickCountByURLID(ctx context.Context, urlID uuid.UUID) (int64, error) {
	querier := r.getQuerier(nil)
	count, err := querier.GetClickCountByURLID(ctx, pgtype.UUID{Bytes: urlID, Valid: true})
	return count, translateError(err, "click")
}

func (r *ClickRepository) GetClicksByUserID(ctx context.Context, dto GetClicksByUserDTO) ([]dbgen.Click, error) {
	querier := r.getQuerier(dto.Tx)
	clicks, err := querier.GetClicksByUserID(ctx, dbgen.GetClicksByUserIDParams{
		UserID: pgtype.UUID{Bytes: dto.UserID, Valid: true},
		Limit:  dto.Limit,
		Offset: dto.Offset,
	})
	return clicks, translateError(err, "click")
}

func (r *ClickRepository) GetClickCountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	querier := r.getQuerier(nil)
	count, err := querier.GetClickCountByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	return count, translateError(err, "click")
}

func (r *ClickRepository) GetClickStatsByURLID(ctx context.Context, urlID uuid.UUID) (dbgen.GetClickStatsByURLIDRow, error) {
	querier := r.getQuerier(nil)
	stats, err := querier.GetClickStatsByURLID(ctx, pgtype.UUID{Bytes: urlID, Valid: true})
	return stats, translateError(err, "click")
}

func (r *ClickRepository) GetDeviceStatsByURLID(ctx context.Context, urlID uuid.UUID) ([]dbgen.GetDeviceStatsByURLIDRow, error) {
	querier := r.getQuerier(nil)
	stats, err := querier.GetDeviceStatsByURLID(ctx, pgtype.UUID{Bytes: urlID, Valid: true})
	return stats, translateError(err, "click")
}

func (r *ClickRepository) GetBrowserStatsByURLID(ctx context.Context, urlID uuid.UUID) ([]dbgen.GetBrowserStatsByURLIDRow, error) {
	querier := r.getQuerier(nil)
	stats, err := querier.GetBrowserStatsByURLID(ctx, pgtype.UUID{Bytes: urlID, Valid: true})
	return stats, translateError(err, "click")
}

func (r *ClickRepository) GetGeoStatsByURLID(ctx context.Context, urlID uuid.UUID) ([]dbgen.GetGeoStatsByURLIDRow, error) {
	querier := r.getQuerier(nil)
	stats, err := querier.GetGeoStatsByURLID(ctx, pgtype.UUID{Bytes: urlID, Valid: true})
	return stats, translateError(err, "click")
}

func (r *ClickRepository) GetClicksByDateRange(ctx context.Context, urlID uuid.UUID, start, end string) ([]dbgen.GetClickStatsByDateRangeRow, error) {
	querier := r.getQuerier(nil)
	stats, err := querier.GetClickStatsByDateRange(ctx, dbgen.GetClickStatsByDateRangeParams{
		UrlID: pgtype.UUID{Bytes: urlID, Valid: true},
	})
	return stats, translateError(err, "click")
}
