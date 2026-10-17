package execution

import (
	"path/filepath"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

// brokenTestFileFixture references a symbol that does not exist, so the test
// package fails to build regardless of the mutant.
const brokenTestFileFixture = `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != undefinedSymbol {
		t.Fatal("unreachable")
	}
}
`

func TestRunSingleMutation_TestBuildFailureIsError(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the Go toolchain")
	}

	t.Parallel()

	dir := newTestModule(t, map[string]string{
		"valid.go":      validGoFileFixture,
		"valid_test.go": brokenTestFileFixture,
	})

	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	mutant := mutation.Mutant{
		ID:       "test-build-failed",
		Type:     "arithmetic_binary",
		FilePath: filepath.Join(dir, "valid.go"),
		Line:     4,
		Column:   9,
		Original: "+",
		Mutated:  "-",
	}

	result := engine.runSingleMutation(mutant, 60)

	if result.Status != mutation.StatusError {
		t.Errorf("expected status %v, got %v\nError: %s\nOutput: %s",
			mutation.StatusError, result.Status, result.Error, result.Output)
	}

	if result.Output == "" {
		t.Error("expected go test output to be captured")
	}
}
