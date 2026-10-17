package execution

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

func TestGoTestArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		extra []string
		want  []string
	}{
		{
			name:  "no extra args",
			extra: nil,
			want:  []string{"test", "-overlay=/tmp/overlay.json", "-vet=off", "."},
		},
		{
			name:  "extra args precede the package",
			extra: []string{"-short", "-run", "^TestUnit"},
			want:  []string{"test", "-overlay=/tmp/overlay.json", "-vet=off", "-short", "-run", "^TestUnit", "."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := goTestArgs("/tmp/overlay.json", tt.extra)
			if !slices.Equal(got, tt.want) {
				t.Errorf("goTestArgs = %v, want %v", got, tt.want)
			}
		})
	}
}

const vetSensitiveFixture = `package main

import "fmt"

func Describe(x int) string {
	return fmt.Sprintf("%d", x)
}

func main() {
	println(Describe(1))
}
`

// vetSensitiveTestFixture exercises Describe without asserting on its output,
// so only vet - not a test - can reject the empty format string.
const vetSensitiveTestFixture = `package main

import "testing"

func TestDescribe(t *testing.T) {
	_ = Describe(1)
}
`

func TestRunSingleMutation_VetFailureIsNotKilled(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the Go toolchain")
	}

	t.Parallel()

	dir := newTestModule(t, map[string]string{
		"main.go":      vetSensitiveFixture,
		"main_test.go": vetSensitiveTestFixture,
	})

	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	mutant := mutation.Mutant{
		ID:       "vet-format",
		Type:     "string_literal",
		FilePath: filepath.Join(dir, "main.go"),
		Line:     6,
		Column:   21,
		Original: `"%d"`,
		Mutated:  `""`,
	}

	result := engine.runSingleMutation(mutant, 60)

	// The test never inspects the string, so no test can catch this mutant.
	// A KILLED here means go test's built-in vet pass rejected the package.
	if result.Status != mutation.StatusSurvived {
		t.Errorf("expected status %v, got %v\nError: %s\nOutput: %s",
			mutation.StatusSurvived, result.Status, result.Error, result.Output)
	}
}
