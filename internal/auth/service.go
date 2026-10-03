package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
	"go.uber.org/zap"
)

type AuthServiceInterface interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	// Logout revokes the given access token and, if non-empty, the refresh token.
	Logout(ctx context.Context, access *Claims, refreshToken string) error
	// Authenticate validates an access token and checks it is not revoked and
	// that its user is still active. The returned claims carry the user's
	// current role, which may differ from the role the token was issued with.
	Authenticate(ctx context.Context, tokenString string) (*Claims, error)
	ValidateToken(tokenString string) (*Claims, error)
	// Refresh rotates a refresh token: it is revoked and a new pair issued.
	Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error)
}

type AuthService struct {
	userRepo UserRepositoryInterface
	store    TokenStore
	env      *infra.Env
	logger   *zap.Logger
}

func NewAuthService(userRepo UserRepositoryInterface, store TokenStore, env *infra.Env, logger *zap.Logger) AuthServiceInterface {
	return &AuthService{
		userRepo: userRepo,
		store:    store,
		env:      env,
		logger:   logger,
	}
}

var errInvalidToken = response.UnauthorizedError{Message: "invalid or expired token"}

func (s *AuthService) generateToken(userID uuid.UUID, email, role, tokenType, secret string, expiryMinutes int) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "url-shortener",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// issueTokens creates a fresh access/refresh token pair for user.
func (s *AuthService) issueTokens(user dbgen.User, role string) (*TokenResponse, error) {
	access, err := s.generateToken(user.ID, user.Email, role, TokenTypeAccess, s.env.AccessTokenSecret, s.env.AccessTokenExpiryMinute)
	if err != nil {
		s.logger.Error("Failed to generate access token", zap.Error(err))
		return nil, response.NewAppError("Failed to generate access token")
	}
	refresh, err := s.generateToken(user.ID, user.Email, role, TokenTypeRefresh, s.env.RefreshTokenSecret, s.env.RefreshTokenExpiryMinute)
	if err != nil {
		s.logger.Error("Failed to generate refresh token", zap.Error(err))
		return nil, response.NewAppError("Failed to generate refresh token")
	}
	return &TokenResponse{Token: access, RefreshToken: refresh}, nil
}

func (s *AuthService) roleName(ctx context.Context, user dbgen.User) string {
	roleName, err := s.userRepo.GetRoleNameByID(ctx, user.RoleID)
	if err != nil {
		s.logger.Warn("Failed to get role name for user, defaulting to 'user'", zap.Error(err))
		return "user"
	}
	return roleName
}

func (s *AuthService) authResponse(user dbgen.User, role string) (*AuthResponse, error) {
	tokens, err := s.issueTokens(user, role)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		Token:        tokens.Token,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      role,
		},
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, response.NewAppError("Failed to process registration")
	}

	tx, err := s.userRepo.GetPool().Begin(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := s.userRepo.CreateUser(ctx, CreateUserDTO{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Tx:           tx,
	})
	if err != nil {
		s.logger.Error("Failed to create user in database", zap.Error(err))
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, err
	}

	return s.authResponse(user, s.roleName(ctx, user))
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, GetUserDTO{Email: &req.Email})
	if err != nil {
		s.logger.Warn("User login failed: email not found", zap.String("email", req.Email))
		return nil, response.NewAppError("Invalid email or password")
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		s.logger.Warn("User login failed: incorrect password", zap.String("email", req.Email))
		return nil, response.NewAppError("Invalid email or password")
	}

	return s.authResponse(user, s.roleName(ctx, user))
}

func (s *AuthService) Logout(ctx context.Context, access *Claims, refreshToken string) error {
	if access != nil {
		s.revoke(ctx, &access.RegisteredClaims)
	}
	if refreshToken == "" {
		return nil
	}
	refresh, err := s.parseToken(refreshToken, s.env.RefreshTokenSecret, TokenTypeRefresh)
	if err != nil {
		// Already invalid or expired: nothing left to revoke.
		return nil
	}
	if access != nil && refresh.UserID != access.UserID {
		return response.PermissionDeniedError{Message: "refresh token belongs to another user"}
	}
	s.revoke(ctx, &refresh.RegisteredClaims)
	return nil
}

// revoke records a token ID as revoked until the token would expire anyway.
func (s *AuthService) revoke(ctx context.Context, claims *jwt.RegisteredClaims) {
	if claims.ExpiresAt == nil {
		return
	}
	if err := s.store.Revoke(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
		s.logger.Error("Failed to revoke token", zap.String("jti", claims.ID), zap.Error(err))
	}
}

// parseToken verifies signature, expiry and token type. Tokens issued before
// token types existed carry no type and are accepted as access tokens.
func (s *AuthService) parseToken(tokenString, secret, wantType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Type != wantType && !(claims.Type == "" && wantType == TokenTypeAccess) {
		return nil, errors.New("wrong token type")
	}
	return claims, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	return s.parseToken(tokenString, s.env.AccessTokenSecret, TokenTypeAccess)
}

func (s *AuthService) Authenticate(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, errInvalidToken
	}
	if err := s.checkNotRevoked(ctx, claims); err != nil {
		return nil, err
	}
	role, err := s.currentRole(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	claims.Role = role
	return claims, nil
}

// checkNotRevoked fails open when the store is unreachable, so a Redis outage
// degrades logout rather than locking every user out.
func (s *AuthService) checkNotRevoked(ctx context.Context, claims *Claims) error {
	revoked, err := s.store.IsRevoked(ctx, claims.ID)
	if err != nil {
		s.logger.Warn("Token revocation check failed, allowing request", zap.Error(err))
		return nil
	}
	if revoked {
		return errInvalidToken
	}
	return nil
}

// currentRole returns the user's role as of now, rejecting deactivated or
// deleted users. Results are cached briefly; the admin API invalidates the
// cache when it changes a user's status or role.
func (s *AuthService) currentRole(ctx context.Context, userID string) (string, error) {
	if role, ok := s.store.CachedRole(ctx, userID); ok {
		if role == inactiveMarker {
			return "", response.UnauthorizedError{Message: "account is deactivated"}
		}
		return role, nil
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return "", errInvalidToken
	}
	// GetUserByID only returns active users.
	user, err := s.userRepo.GetUserByID(ctx, GetUserDTO{ID: &id})
	if err != nil {
		var notFound response.NotFoundError
		if errors.As(err, &notFound) {
			s.store.CacheRole(ctx, userID, inactiveMarker)
			return "", response.UnauthorizedError{Message: "account is deactivated"}
		}
		return "", err
	}
	role := s.roleName(ctx, user)
	s.store.CacheRole(ctx, userID, role)
	return role, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	claims, err := s.parseToken(refreshToken, s.env.RefreshTokenSecret, TokenTypeRefresh)
	if err != nil {
		return nil, errInvalidToken
	}
	revoked, err := s.store.IsRevoked(ctx, claims.ID)
	if err != nil {
		s.logger.Warn("Refresh token revocation check failed", zap.Error(err))
	}
	if revoked {
		s.logger.Warn("Revoked refresh token presented", zap.String("user_id", claims.UserID))
		return nil, errInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errInvalidToken
	}
	user, err := s.userRepo.GetUserByID(ctx, GetUserDTO{ID: &userID})
	if err != nil {
		var notFound response.NotFoundError
		if errors.As(err, &notFound) {
			return nil, response.UnauthorizedError{Message: "account is deactivated"}
		}
		return nil, err
	}

	// Rotate: the presented refresh token can't be used again.
	s.revoke(ctx, &claims.RegisteredClaims)
	return s.issueTokens(user, s.roleName(ctx, user))
}
