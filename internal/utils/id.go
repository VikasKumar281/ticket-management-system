package utils

import (
	"crypto/rand"
	"fmt"
)

// NewID generates a random UUID (v4-style) string using crypto/rand.
// We avoid pulling in the google/uuid dependency to keep this service
// fully self-contained (no external modules, no network access needed
// to build/deploy the Docker image).
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read failing is effectively impossible on any
		// supported platform; panic is acceptable here since it would
		// indicate a broken runtime environment.
		panic("failed to generate random id: " + err.Error())
	}

	// Set version (4) and variant (RFC 4122) bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
