package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
	"go.uber.org/zap"
)

type AuthServiceInterface interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(ctx context.Context, userID uuid.UUID) (string, error)
}

type AuthService struct {
	userRepo UserRepositoryInterface
	env      *infra.Env
	logger   *zap.Logger
}

func NewAuthService(userRepo UserRepositoryInterface, env *infra.Env, logger *zap.Logger) AuthServiceInterface {
	return &AuthService{
		userRepo: userRepo,
		env:      env,
		logger:   logger,
	}
}

func (s *AuthService) generateToken(userID uuid.UUID, email string, role string, secret string, expiryMinutes int) (string, error) {
	claims := Claims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "url-shortener",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
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
	defer tx.Rollback(ctx)

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

	roleName, err := s.userRepo.GetRoleNameByID(ctx, user.RoleID)
	if err != nil {
		s.logger.Warn("Failed to get role name for user, defaulting to 'user'", zap.Error(err))
		roleName = "user"
	}

	token, err := s.generateToken(user.ID, user.Email, roleName, s.env.AccessTokenSecret, s.env.AccessTokenExpiryMinute)
	if err != nil {
		s.logger.Error("Failed to generate access token during registration", zap.Error(err))
		return nil, response.NewAppError("Failed to generate access token")
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
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

	roleName, err := s.userRepo.GetRoleNameByID(ctx, user.RoleID)
	if err != nil {
		s.logger.Warn("Failed to get role name for user during login, defaulting to 'user'", zap.Error(err))
		roleName = "user"
	}

	token, err := s.generateToken(user.ID, user.Email, roleName, s.env.AccessTokenSecret, s.env.AccessTokenExpiryMinute)
	if err != nil {
		s.logger.Error("Failed to generate access token during login", zap.Error(err))
		return nil, response.NewAppError("Failed to generate access token")
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetUserByID(ctx, GetUserDTO{ID: &userID})
	if err != nil {
		s.logger.Warn("User logout failed: user not found", zap.String("userID", userID.String()))
		return response.NewAppError("User not found")
	}

	roleName, err := s.userRepo.GetRoleNameByID(ctx, user.RoleID)
	if err != nil {
		s.logger.Warn("Failed to get role name for user during logout, defaulting to 'user'", zap.Error(err))
		roleName = "user"
	}

	_, err = s.generateToken(user.ID, user.Email, roleName, s.env.AccessTokenSecret, s.env.AccessTokenExpiryMinute)
	if err != nil {
		s.logger.Error("Failed to generate access token during logout", zap.Error(err))
		return response.NewAppError("Failed to generate access token")
	}

	return nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.env.AccessTokenSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func (s *AuthService) RefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := s.userRepo.GetUserByID(ctx, GetUserDTO{ID: &userID})
	if err != nil {
		s.logger.Warn("Token refresh failed: user not found", zap.String("userID", userID.String()))
		return "", response.NewAppError("User not found")
	}

	roleName, err := s.userRepo.GetRoleNameByID(ctx, user.RoleID)
	if err != nil {
		s.logger.Warn("Failed to get role name for user during token refresh, defaulting to 'user'", zap.Error(err))
		roleName = "user"
	}

	token, err := s.generateToken(user.ID, user.Email, roleName, s.env.AccessTokenSecret, s.env.AccessTokenExpiryMinute)
	if err != nil {
		s.logger.Error("Failed to generate access token during token refresh", zap.Error(err))
		return "", response.NewAppError("Failed to generate access token")
	}

	return token, nil
}
