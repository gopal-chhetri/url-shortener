package auth

import (
	"github.com/golang-jwt/jwt/v4"
)

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email" example:"user@gmail.com"`
	Password  string `json:"password" binding:"required,min=6" example:"password123"`
	FirstName string `json:"first_name" binding:"required" example:"User"`
	LastName  string `json:"last_name" binding:"required" example:"Name"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"user@gmail.com" binding:"required,email"`
	Password string `json:"password" example:"password123" binding:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
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
