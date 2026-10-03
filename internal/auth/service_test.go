package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
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
	token, err := svc.generateToken(userID, "test@example.com", "user", TokenTypeAccess, "test-secret", 60)

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
	token, err := svc.generateToken(userID, "test@example.com", "user", TokenTypeAccess, env.AccessTokenSecret, 60)
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

// memTokenStore is an in-memory TokenStore for tests.
type memTokenStore struct {
	revoked map[string]bool
	roles   map[string]string
}

func newMemTokenStore() *memTokenStore {
	return &memTokenStore{revoked: map[string]bool{}, roles: map[string]string{}}
}

func (m *memTokenStore) Revoke(_ context.Context, jti string, _ time.Time) error {
	m.revoked[jti] = true
	return nil
}

func (m *memTokenStore) IsRevoked(_ context.Context, jti string) (bool, error) {
	return m.revoked[jti], nil
}

func (m *memTokenStore) CachedRole(_ context.Context, userID string) (string, bool) {
	r, ok := m.roles[userID]
	return r, ok
}

func (m *memTokenStore) CacheRole(_ context.Context, userID, role string) {
	m.roles[userID] = role
}

// newRefreshTestService returns a service whose single user is active until
// *active is set to false.
func newRefreshTestService(userID uuid.UUID, active *bool) *AuthService {
	repo := &mockUserRepository{
		getUserByIDFn: func(ctx context.Context, dto GetUserDTO) (dbgen.User, error) {
			if !*active {
				return dbgen.User{}, response.NotFoundError{Model: "user"}
			}
			return dbgen.User{ID: userID, Email: "test@example.com", RoleID: uuid.New()}, nil
		},
		getRoleNameByIDFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return "admin", nil
		},
	}
	return &AuthService{userRepo: repo, store: newMemTokenStore(), env: newTestEnv(), logger: newTestLogger()}
}

func TestRefresh_RotatesTokens(t *testing.T) {
	userID := uuid.New()
	active := true
	svc := newRefreshTestService(userID, &active)
	ctx := context.Background()

	refresh, err := svc.generateToken(userID, "test@example.com", "user", TokenTypeRefresh, svc.env.RefreshTokenSecret, 60)
	require.NoError(t, err)

	tokens, err := svc.Refresh(ctx, refresh)
	require.NoError(t, err)
	claims, err := svc.Authenticate(ctx, tokens.Token)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)

	// The old refresh token was rotated out.
	_, err = svc.Refresh(ctx, refresh)
	var unauth response.UnauthorizedError
	assert.ErrorAs(t, err, &unauth)

	// The new one works.
	_, err = svc.Refresh(ctx, tokens.RefreshToken)
	assert.NoError(t, err)
}

func TestRefresh_RejectsAccessToken(t *testing.T) {
	userID := uuid.New()
	active := true
	svc := newRefreshTestService(userID, &active)

	access, err := svc.generateToken(userID, "test@example.com", "user", TokenTypeAccess, svc.env.AccessTokenSecret, 60)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), access)
	var unauth response.UnauthorizedError
	assert.ErrorAs(t, err, &unauth)
}

func TestAuthenticate_RejectsRefreshToken(t *testing.T) {
	userID := uuid.New()
	active := true
	svc := newRefreshTestService(userID, &active)

	refresh, err := svc.generateToken(userID, "test@example.com", "user", TokenTypeRefresh, svc.env.RefreshTokenSecret, 60)
	require.NoError(t, err)

	_, err = svc.Authenticate(context.Background(), refresh)
	assert.Error(t, err)
}

func TestLogout_RevokesBothTokens(t *testing.T) {
	userID := uuid.New()
	active := true
	svc := newRefreshTestService(userID, &active)
	ctx := context.Background()

	pair, err := svc.issueTokens(dbgen.User{ID: userID, Email: "test@example.com"}, "user")
	require.NoError(t, err)
	claims, err := svc.Authenticate(ctx, pair.Token)
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, claims, pair.RefreshToken))

	_, err = svc.Authenticate(ctx, pair.Token)
	assert.Error(t, err, "access token must be revoked")
	_, err = svc.Refresh(ctx, pair.RefreshToken)
	assert.Error(t, err, "refresh token must be revoked")
}

func TestAuthenticate_DeactivatedUser(t *testing.T) {
	userID := uuid.New()
	active := true
	svc := newRefreshTestService(userID, &active)
	ctx := context.Background()

	access, err := svc.generateToken(userID, "test@example.com", "user", TokenTypeAccess, svc.env.AccessTokenSecret, 60)
	require.NoError(t, err)

	active = false
	_, err = svc.Authenticate(ctx, access)
	var unauth response.UnauthorizedError
	assert.ErrorAs(t, err, &unauth)
}

func hashPassword(password string) (string, error) {
	return utils.HashPassword(password)
}
