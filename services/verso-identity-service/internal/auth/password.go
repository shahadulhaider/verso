// Package auth provides password hashing and JWT token management.
package auth

import "github.com/alexedwards/argon2id"

// HashPassword produces an Argon2id hash of the given plaintext password.
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

// VerifyPassword checks whether a plaintext password matches the given hash.
func VerifyPassword(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
