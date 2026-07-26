// Package idgen provides a minimal, dependency-free unique ID generator.
// In a real project you'd likely still reach for github.com/google/uuid,
// but this keeps the example runnable with zero external dependencies.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a random 16-byte hex-encoded ID, e.g. "a3f9c2e1b7d84f0a91c6e2b5d4f7a8c1".
func New() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// crypto/rand.Read only fails in extreme, practically unreachable
		// conditions (OS entropy source unavailable) — panicking here is
		// standard practice, same as Go's own crypto packages do.
		panic("idgen: failed to generate random ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
