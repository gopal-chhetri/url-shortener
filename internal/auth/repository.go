package auth

import (
	"context"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateUserDTO struct {
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Tx           pgx.Tx `json:"-"`
}

type UpdateUserDTO struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Tx        pgx.Tx    `json:"-"`
}

type GetUserDTO struct {
	ID    *uuid.UUID `json:"id,omitempty"`
	Email *string    `json:"email,omitempty"`
	Tx    pgx.Tx     `json:"-"`
}

type ListUsersDTO struct {
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
	Tx     pgx.Tx `json:"-"`
}

type DeleteUserDTO struct {
	ID uuid.UUID `json:"id"`
	Tx pgx.Tx    `json:"-"`
}

type UserRepositoryInterface interface {
	CreateUser(ctx context.Context, dto CreateUserDTO) (dbgen.User, error)
	GetUserByEmail(ctx context.Context, dto GetUserDTO) (dbgen.User, error)
	GetUserByID(ctx context.Context, dto GetUserDTO) (dbgen.User, error)
	UpdateUser(ctx context.Context, dto UpdateUserDTO) (dbgen.User, error)
	DeleteUser(ctx context.Context, dto DeleteUserDTO) error
	ListUsers(ctx context.Context, dto ListUsersDTO) ([]dbgen.User, error)
	CountUsers(ctx context.Context, dto struct{ Tx pgx.Tx }) (int64, error)
	GetRoleByName(ctx context.Context, name string) (dbgen.Role, error)
	GetRoleNameByID(ctx context.Context, id uuid.UUID) (string, error)
	GetPool() *pgxpool.Pool
}

type UserRepository struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepositoryInterface {
	return &UserRepository{
		pool:    pool,
		queries: dbgen.New(pool),
	}
}

func (r *UserRepository) GetPool() *pgxpool.Pool {
	return r.pool
}

func (r *UserRepository) getQuerier(tx pgx.Tx) *dbgen.Queries {
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

func (r *UserRepository) CreateUser(ctx context.Context, dto CreateUserDTO) (dbgen.User, error) {
	querier := r.getQuerier(dto.Tx)
	user, err := querier.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        dto.Email,
		PasswordHash: dto.PasswordHash,
		FirstName:    dto.FirstName,
		LastName:     dto.LastName,
	})
	return user, translateError(err, "user")
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
	querier := r.getQuerier(dto.Tx)
	user, err := querier.GetUserByEmail(ctx, *dto.Email)
	return user, translateError(err, "user")
}

func (r *UserRepository) GetUserByID(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
	querier := r.getQuerier(dto.Tx)
	user, err := querier.GetUserById(ctx, *dto.ID)
	return user, translateError(err, "user")
}

func (r *UserRepository) UpdateUser(ctx context.Context, dto UpdateUserDTO) (dbgen.User, error) {
	querier := r.getQuerier(dto.Tx)
	user, err := querier.UpdateUserById(ctx, dbgen.UpdateUserByIdParams{
		ID:        dto.ID,
		FirstName: dto.FirstName,
		LastName:  dto.LastName,
		Email:     "", // we can pass empty string or support email update if needed
	})
	return user, translateError(err, "user")
}

func (r *UserRepository) DeleteUser(ctx context.Context, dto DeleteUserDTO) error {
	querier := r.getQuerier(dto.Tx)
	return querier.DeleteUserById(ctx, dto.ID)
}

func (r *UserRepository) ListUsers(ctx context.Context, dto ListUsersDTO) ([]dbgen.User, error) {
	querier := r.getQuerier(dto.Tx)
	return querier.ListUsers(ctx, dbgen.ListUsersParams{
		Limit:  dto.Limit,
		Offset: dto.Offset,
	})
}

func (r *UserRepository) CountUsers(ctx context.Context, dto struct{ Tx pgx.Tx }) (int64, error) {
	querier := r.getQuerier(dto.Tx)
	return querier.CountUsers(ctx)
}

func (r *UserRepository) GetRoleByName(ctx context.Context, name string) (dbgen.Role, error) {
	querier := r.getQuerier(nil)
	role, err := querier.GetRoleByName(ctx, name)
	return role, translateError(err, "role")
}

func (r *UserRepository) GetRoleNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	querier := r.getQuerier(nil)
	name, err := querier.GetRoleNameByID(ctx, id)
	return name, translateError(err, "role")
}
