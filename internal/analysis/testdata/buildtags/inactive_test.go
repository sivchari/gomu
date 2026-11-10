//go:build !unit

package example_test

import (
	"testing"

	example "example.com/gomu-build-repro"
)

func TestValue(t *testing.T) {
	if got := example.Value(); got != 30 {
		t.Fatalf("Value() = %d, want 30", got)
	}
}
