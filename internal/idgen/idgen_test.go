package idgen

import (
	"regexp"
	"testing"
)

// TestNew checks that idgen.New() generates valid unique IDs
func TestNew(t *testing.T) {
	// Generate two IDs
	id1 := New()
	id2 := New()

	// Test 1: ID should be 32 characters (16 bytes hex-encoded)
	if len(id1) != 32 {
		t.Errorf("expected ID length 32, got %d", len(id1))
	}

	// Test 2: Multiple calls should generate different IDs
	if id1 == id2 {
		t.Errorf("expected different IDs, got same: %s", id1)
	}

	// Test 3: IDs should only contain valid hex characters
	hexPattern := regexp.MustCompile("^[0-9a-f]+$")
	if !hexPattern.MatchString(id1) {
		t.Errorf("ID contains non-hex characters: %s", id1)
	}
	if !hexPattern.MatchString(id2) {
		t.Errorf("ID contains non-hex characters: %s", id2)
	}
}