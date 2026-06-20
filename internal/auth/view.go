package auth

import (
	"github.com/golang-jwt/jwt/v4"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
)

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email" example:"johndoe@gmail.com"`
	Password  string `json:"password" binding:"required,min=6" example:"password123"`
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" binding:"required" example:"Doe"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"johndoe@gmail.com" binding:"required,email"`
	Password string `json:"password" example:"password123" binding:"required"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  dbgen.User `json:"user"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenResponse struct {
	Token string `json:"token" example:"jwt_token_string"`
}
