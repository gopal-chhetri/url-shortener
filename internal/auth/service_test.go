package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockUserRepository struct {
	createUserFn      func(ctx context.Context, dto CreateUserDTO) (dbgen.User, error)
	getUserByEmailFn  func(ctx context.Context, dto GetUserDTO) (dbgen.User, error)
	getUserByIDFn     func(ctx context.Context, dto GetUserDTO) (dbgen.User, error)
	getRoleNameByIDFn func(ctx context.Context, id uuid.UUID) (string, error)
	getPoolFn         func() *pgxpool.Pool
}

func (m *mockUserRepository) CreateUser(ctx context.Context, dto CreateUserDTO) (dbgen.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, dto)
	}
	return dbgen.User{}, nil
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, dto)
	}
	return dbgen.User{}, nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, dto)
	}
	return dbgen.User{}, nil
}

func (m *mockUserRepository) UpdateUser(ctx context.Context, dto UpdateUserDTO) (dbgen.User, error) {
	return dbgen.User{}, nil
}

func (m *mockUserRepository) DeleteUser(ctx context.Context, dto DeleteUserDTO) error {
	return nil
}

func (m *mockUserRepository) ListUsers(ctx context.Context, dto ListUsersDTO) ([]dbgen.User, error) {
	return nil, nil
}

func (m *mockUserRepository) CountUsers(ctx context.Context, dto struct{ Tx pgx.Tx }) (int64, error) {
	return 0, nil
}

func (m *mockUserRepository) GetRoleByName(ctx context.Context, name string) (dbgen.Role, error) {
	return dbgen.Role{}, nil
}

func (m *mockUserRepository) GetRoleNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	if m.getRoleNameByIDFn != nil {
		return m.getRoleNameByIDFn(ctx, id)
	}
	return "user", nil
}

func (m *mockUserRepository) GetPool() *pgxpool.Pool {
	if m.getPoolFn != nil {
		return m.getPoolFn()
	}
	return nil
}

func newTestEnv() *infra.Env {
	return &infra.Env{
		AccessTokenSecret:        "test-secret-key-for-testing-12345678",
		AccessTokenExpiryMinute:  60,
		RefreshTokenSecret:       "test-refresh-secret-key-for-testing",
		RefreshTokenExpiryMinute: 10080,
	}
}

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestGenerateToken(t *testing.T) {
	svc := &AuthService{
		env:    newTestEnv(),
		logger: newTestLogger(),
	}

	userID := uuid.New()
	token, err := svc.generateToken(userID, "test@example.com", "user", "test-secret", 60)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Valid(t *testing.T) {
	env := newTestEnv()
	svc := &AuthService{
		env:    env,
		logger: newTestLogger(),
	}

	userID := uuid.New()
	token, err := svc.generateToken(userID, "test@example.com", "user", env.AccessTokenSecret, 60)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "user", claims.Role)
}

func TestValidateToken_Expired(t *testing.T) {
	env := newTestEnv()
	svc := &AuthService{
		env:    env,
		logger: newTestLogger(),
	}

	claims := Claims{
		UserID: uuid.New().String(),
		Email:  "test@example.com",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "url-shortener",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(env.AccessTokenSecret))
	require.NoError(t, err)

	result, err := svc.ValidateToken(tokenString)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := &AuthService{
		env:    newTestEnv(),
		logger: newTestLogger(),
	}

	result, err := svc.ValidateToken("invalid-token-string")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLogin_Success(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	hashedPassword, err := hashPassword("password123")
	require.NoError(t, err)

	mockRepo := &mockUserRepository{
		getUserByEmailFn: func(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
			return dbgen.User{
				ID:           userID,
				Email:        *dto.Email,
				PasswordHash: hashedPassword,
				FirstName:    "Test",
				LastName:     "User",
				RoleID:       roleID,
			}, nil
		},
		getRoleNameByIDFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return "user", nil
		},
	}

	svc := &AuthService{
		userRepo: mockRepo,
		env:      newTestEnv(),
		logger:   newTestLogger(),
	}

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "test@example.com", resp.User.Email)
}

func TestLogin_InvalidEmail(t *testing.T) {
	mockRepo := &mockUserRepository{
		getUserByEmailFn: func(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
			return dbgen.User{}, errors.New("not found")
		},
	}

	svc := &AuthService{
		userRepo: mockRepo,
		env:      newTestEnv(),
		logger:   newTestLogger(),
	}

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var appErr response.AppError
	assert.True(t, errors.As(err, &appErr))
}

func TestLogin_WrongPassword(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	hashedPassword, _ := hashPassword("correctpassword")

	mockRepo := &mockUserRepository{
		getUserByEmailFn: func(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
			return dbgen.User{
				ID:           userID,
				Email:        *dto.Email,
				PasswordHash: hashedPassword,
				FirstName:    "Test",
				LastName:     "User",
				RoleID:       roleID,
			}, nil
		},
	}

	svc := &AuthService{
		userRepo: mockRepo,
		env:      newTestEnv(),
		logger:   newTestLogger(),
	}

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var appErr response.AppError
	assert.True(t, errors.As(err, &appErr))
}

func TestRefreshToken_Success(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()

	mockRepo := &mockUserRepository{
		getUserByIDFn: func(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
			return dbgen.User{
				ID:        userID,
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				RoleID:    roleID,
			}, nil
		},
		getRoleNameByIDFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return "admin", nil
		},
	}

	svc := &AuthService{
		userRepo: mockRepo,
		env:      newTestEnv(),
		logger:   newTestLogger(),
	}

	token, err := svc.RefreshToken(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	mockRepo := &mockUserRepository{
		getUserByIDFn: func(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
			return dbgen.User{}, errors.New("not found")
		},
	}

	svc := &AuthService{
		userRepo: mockRepo,
		env:      newTestEnv(),
		logger:   newTestLogger(),
	}

	token, err := svc.RefreshToken(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Empty(t, token)

	var appErr response.AppError
	assert.True(t, errors.As(err, &appErr))
}

func hashPassword(password string) (string, error) {
	return utils.HashPassword(password)
}
