package utils

import "golang.org/x/crypto/bcrypt"

// passwordHashCost is the bcrypt work factor. 12 keeps a login well under
// the cost of 14 (~1s CPU each), which made /auth/login a cheap DoS target.
// Existing hashes keep verifying: bcrypt stores the cost inside each hash.
const passwordHashCost = 12

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), passwordHashCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
