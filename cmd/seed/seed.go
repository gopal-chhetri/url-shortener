package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
)

func main() {
	fmt.Println("Initializing seeder...")
	app := bootstrap.NewApplication()
	defer app.Database.Close()

	ctx := context.Background()
	pool := app.Database.GetPool()

	// Hash password for default users
	hashedPassword, err := utils.HashPassword("password123")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	fmt.Println("Seeding users...")

	// 1. Get role IDs
	var adminRoleID string
	err = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'admin'").Scan(&adminRoleID)
	if err != nil {
		log.Fatalf("Admin role not found (have migrations run?): %v", err)
	}

	var userRoleID string
	err = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'user'").Scan(&userRoleID)
	if err != nil {
		log.Fatalf("User role not found: %v", err)
	}

	// 2. Insert Admin User
	_, err = pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, first_name, last_name, role_id, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (email) DO NOTHING
	`, "admin@gmail.com", hashedPassword, "Admin", "System", adminRoleID)
	if err != nil {
		log.Fatalf("Failed to seed admin user: %v", err)
	}

	// 3. Insert Regular User
	_, err = pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, first_name, last_name, role_id, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (email) DO NOTHING
	`, "user@gmail.com", hashedPassword, "Regular", "User", userRoleID)
	if err != nil {
		log.Fatalf("Failed to seed regular user: %v", err)
	}

	fmt.Println("Seeding URLs...")

	// 4. Retrieve regular user's ID
	var regularUserID string
	err = pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'user@gmail.com'").Scan(&regularUserID)
	if err != nil {
		log.Fatalf("Failed to retrieve regular user ID: %v", err)
	}

	// 5. Seed some sample URLs
	sampleURLs := []struct {
		ShortURL    string
		OriginalURL string
	}{
		{"google", "https://www.google.com"},
		{"github", "https://github.com/gopal-chhetri/url-shortener"},
	}

	for _, sample := range sampleURLs {
		_, err = pool.Exec(ctx, `
			INSERT INTO urls (short_url, original_url, user_id, is_active)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (short_url) DO NOTHING
		`, sample.ShortURL, sample.OriginalURL, regularUserID)
		if err != nil {
			log.Printf("Failed to seed URL %s: %v", sample.ShortURL, err)
		}
	}

	fmt.Println("Database seeded successfully!")
}
