package admin

import (
	"context"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepositoryInterface interface {
	// User management
	ListAllUsers(ctx context.Context, limit, offset int32) ([]dbgen.User, error)
	CountAllUsers(ctx context.Context) (int64, error)
	GetUsersByIDs(ctx context.Context, userIDs []uuid.UUID) ([]dbgen.User, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) (dbgen.User, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) (dbgen.User, error)
	GetRoleByName(ctx context.Context, name string) (dbgen.Role, error)
	ListRoles(ctx context.Context) ([]dbgen.Role, error)

	// URL management
	ListAllURLs(ctx context.Context, limit, offset int32) ([]dbgen.Url, error)
	CountAllURLs(ctx context.Context) (int64, error)
	CountActiveURLs(ctx context.Context) (int64, error)
	CountInactiveURLs(ctx context.Context) (int64, error)
	GetTopURLsByClicks(ctx context.Context, limit int32) ([]dbgen.GetTopURLsByClicksRow, error)
	UpdateURLStatus(ctx context.Context, urlID uuid.UUID, isActive bool) (dbgen.Url, error)
	SearchURLs(ctx context.Context, search string, limit, offset int32) ([]dbgen.Url, error)
	CountSearchURLs(ctx context.Context, search string) (int64, error)
	ListAllURLsAdmin(ctx context.Context, limit, offset int32) ([]dbgen.Url, error)
	ListAllURLsAdminByName(ctx context.Context, limit, offset int32) ([]dbgen.Url, error)
	CountAllURLsAdmin(ctx context.Context) (int64, error)
	ListURLsByClicks(ctx context.Context, limit, offset int32) ([]dbgen.ListURLsByClicksRow, error)
	GetClickCountsByURLIDs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	DeleteURL(ctx context.Context, urlID uuid.UUID) error

	// Click stats
	CountAllClicks(ctx context.Context) (int64, error)
}

type AdminRepository struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func NewAdminRepository(pool *pgxpool.Pool) AdminRepositoryInterface {
	return &AdminRepository{
		pool:    pool,
		queries: dbgen.New(pool),
	}
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

func (r *AdminRepository) ListAllUsers(ctx context.Context, limit, offset int32) ([]dbgen.User, error) {
	users, err := r.queries.ListAllUsers(ctx, dbgen.ListAllUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	return users, translateError(err, "user")
}

func (r *AdminRepository) CountAllUsers(ctx context.Context) (int64, error) {
	return r.queries.CountAllUsers(ctx)
}

func (r *AdminRepository) GetUsersByIDs(ctx context.Context, userIDs []uuid.UUID) ([]dbgen.User, error) {
	if len(userIDs) == 0 {
		return []dbgen.User{}, nil
	}

	users, err := r.queries.GetUsersByIDs(ctx, userIDs)
	return users, translateError(err, "user")
}

func (r *AdminRepository) UpdateUserRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) (dbgen.User, error) {
	user, err := r.queries.UpdateUserRole(ctx, dbgen.UpdateUserRoleParams{
		ID:     userID,
		RoleID: roleID,
	})
	return user, translateError(err, "user")
}

func (r *AdminRepository) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) (dbgen.User, error) {
	user, err := r.queries.UpdateUserStatus(ctx, dbgen.UpdateUserStatusParams{
		ID:       userID,
		IsActive: pgtype.Bool{Bool: isActive, Valid: true},
	})
	return user, translateError(err, "user")
}

func (r *AdminRepository) GetRoleByName(ctx context.Context, name string) (dbgen.Role, error) {
	role, err := r.queries.GetRoleByName(ctx, name)
	return role, translateError(err, "role")
}

func (r *AdminRepository) ListRoles(ctx context.Context) ([]dbgen.Role, error) {
	return r.queries.ListRoles(ctx, dbgen.ListRolesParams{
		Limit:  100,
		Offset: 0,
	})
}

func (r *AdminRepository) ListAllURLs(ctx context.Context, limit, offset int32) ([]dbgen.Url, error) {
	urls, err := r.queries.ListAllURLs(ctx, dbgen.ListAllURLsParams{
		Limit:  limit,
		Offset: offset,
	})
	return urls, translateError(err, "url")
}

func (r *AdminRepository) CountAllURLs(ctx context.Context) (int64, error) {
	return r.queries.CountAllURLs(ctx)
}

func (r *AdminRepository) CountActiveURLs(ctx context.Context) (int64, error) {
	return r.queries.CountActiveURLs(ctx)
}

func (r *AdminRepository) CountInactiveURLs(ctx context.Context) (int64, error) {
	return r.queries.CountInactiveURLs(ctx)
}

func (r *AdminRepository) GetTopURLsByClicks(ctx context.Context, limit int32) ([]dbgen.GetTopURLsByClicksRow, error) {
	return r.queries.GetTopURLsByClicks(ctx, limit)
}

func (r *AdminRepository) UpdateURLStatus(ctx context.Context, urlID uuid.UUID, isActive bool) (dbgen.Url, error) {
	url, err := r.queries.UpdateURLStatus(ctx, dbgen.UpdateURLStatusParams{
		ID:       urlID,
		IsActive: pgtype.Bool{Bool: isActive, Valid: true},
	})
	return url, translateError(err, "url")
}

func (r *AdminRepository) CountAllClicks(ctx context.Context) (int64, error) {
	return r.queries.CountAllClicks(ctx)
}

func (r *AdminRepository) SearchURLs(ctx context.Context, search string, limit, offset int32) ([]dbgen.Url, error) {
	urls, err := r.queries.SearchURLs(ctx, dbgen.SearchURLsParams{
		ShortUrl: "%" + search + "%",
		Limit:    limit,
		Offset:   offset,
	})
	return urls, translateError(err, "url")
}

func (r *AdminRepository) CountSearchURLs(ctx context.Context, search string) (int64, error) {
	return r.queries.CountSearchURLs(ctx, "%"+search+"%")
}

func (r *AdminRepository) ListAllURLsAdminByName(ctx context.Context, limit, offset int32) ([]dbgen.Url, error) {
	urls, err := r.queries.ListAllURLsAdminByName(ctx, dbgen.ListAllURLsAdminByNameParams{
		Limit:  limit,
		Offset: offset,
	})
	return urls, translateError(err, "url")
}

func (r *AdminRepository) GetClickCountsByURLIDs(ctx context.Context, urlIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
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

func (r *AdminRepository) DeleteURL(ctx context.Context, urlID uuid.UUID) error {
	err := r.queries.DeleteURL(ctx, urlID)
	return translateError(err, "url")
}

func (r *AdminRepository) ListAllURLsAdmin(ctx context.Context, limit, offset int32) ([]dbgen.Url, error) {
	urls, err := r.queries.ListAllURLsAdmin(ctx, dbgen.ListAllURLsAdminParams{
		Limit:  limit,
		Offset: offset,
	})
	return urls, translateError(err, "url")
}

func (r *AdminRepository) CountAllURLsAdmin(ctx context.Context) (int64, error) {
	return r.queries.CountAllURLsAdmin(ctx)
}

func (r *AdminRepository) ListURLsByClicks(ctx context.Context, limit, offset int32) ([]dbgen.ListURLsByClicksRow, error) {
	urls, err := r.queries.ListURLsByClicks(ctx, dbgen.ListURLsByClicksParams{
		Limit:  limit,
		Offset: offset,
	})
	return urls, translateError(err, "url")
}
