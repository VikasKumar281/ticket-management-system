// Package auth contains password hashing and JWT helpers for the ticket
// system. Everything here is implemented with the Go standard library only
// (crypto/hmac, crypto/sha256, crypto/rand, crypto/subtle) so the service
// has zero third-party dependencies. That keeps `go build` / `docker build`
// fully reliable with no module-proxy network access required — a
// deliberate trade-off for a small, time-boxed assignment. In a larger
// production system you would typically pull in golang.org/x/crypto/bcrypt
// instead of hand-rolling PBKDF2.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	pbkdf2Iterations = 100_000
	pbkdf2KeyLen     = 32
	saltLen          = 16
)

// ErrInvalidHash is returned when a stored password hash is malformed.
var ErrInvalidHash = errors.New("auth: invalid password hash format")

// HashPassword derives a salted PBKDF2-HMAC-SHA256 hash for the given
// plaintext password and returns it encoded as:
//
//	pbkdf2-sha256$<iterations>$<salt-hex>$<hash-hex>
//
// The encoded string is safe to store directly in the database/store.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: failed to generate salt: %w", err)
	}

	derived := pbkdf2(password, salt, pbkdf2Iterations, pbkdf2KeyLen)

	encoded := fmt.Sprintf(
		"pbkdf2-sha256$%d$%s$%s",
		pbkdf2Iterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(derived),
	)
	return encoded, nil
}

// VerifyPassword checks a plaintext password against a hash previously
// produced by HashPassword. It returns true only on an exact match, using
// a constant-time comparison to avoid timing side-channels.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false, ErrInvalidHash
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false, ErrInvalidHash
	}

	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false, ErrInvalidHash
	}

	expected, err := hex.DecodeString(parts[3])
	if err != nil {
		return false, ErrInvalidHash
	}

	actual := pbkdf2(password, salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

// pbkdf2 implements RFC 2898 PBKDF2 using HMAC-SHA256 as the pseudorandom
// function. It is a minimal, dependency-free implementation sufficient for
// password hashing at rest.
func pbkdf2(password string, salt []byte, iterations, keyLen int) []byte {
	const hashLen = sha256.Size

	numBlocks := (keyLen + hashLen - 1) / hashLen
	derived := make([]byte, 0, numBlocks*hashLen)

	mac := hmac.New(sha256.New, []byte(password))

	for block := 1; block <= numBlocks; block++ {
		mac.Reset()
		mac.Write(salt)
		blockIndex := make([]byte, 4)
		binary.BigEndian.PutUint32(blockIndex, uint32(block))
		mac.Write(blockIndex)
		u := mac.Sum(nil)

		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iterations; i++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}

		derived = append(derived, t...)
	}

	return derived[:keyLen]
}
