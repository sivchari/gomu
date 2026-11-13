package execution

import (
	"testing"
	"time"

	"github.com/sivchari/gomu/internal/mutation"
)

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		timeout int
		wantErr string
	}{
		{name: "minimum values", workers: 1, timeout: 1},
		{name: "default values", workers: 4, timeout: 30},
		{name: "zero workers", workers: 0, timeout: 5, wantErr: "workers must be at least 1 (got 0)"},
		{name: "negative workers", workers: -1, timeout: 5, wantErr: "workers must be at least 1 (got -1)"},
		{name: "zero timeout", workers: 1, timeout: 0, wantErr: "timeout must be at least 1 second (got 0)"},
		{name: "negative timeout", workers: 1, timeout: -1, wantErr: "timeout must be at least 1 second (got -1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOptions(tt.workers, tt.timeout)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}

			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunMutationsWithOptionsRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		timeout int
	}{
		{name: "zero workers", workers: 0, timeout: 5},
		{name: "negative workers", workers: -1, timeout: 5},
		{name: "zero timeout", workers: 1, timeout: 0},
		{name: "negative timeout", workers: 1, timeout: -1},
	}
	mutantSets := []struct {
		name    string
		mutants []mutation.Mutant
	}{
		{name: "nil mutants"},
		{name: "empty mutants", mutants: []mutation.Mutant{}},
		{name: "nonempty mutants", mutants: []mutation.Mutant{{ID: "invalid-options"}}},
	}

	for _, tt := range tests {
		for _, set := range mutantSets {
			t.Run(tt.name+"/"+set.name, func(t *testing.T) {
				type outcome struct {
					results []mutation.Result
					err     error
				}

				done := make(chan outcome, 1)

				go func() {
					// Invalid options must be rejected before accessing any engine resources.
					engine := &Engine{}

					results, err := engine.RunMutationsWithOptions(set.mutants, tt.workers, tt.timeout)
					done <- outcome{results: results, err: err}
				}()

				select {
				case got := <-done:
					if got.err == nil {
						t.Fatal("expected invalid options to return an error")
					}

					if got.results != nil {
						t.Fatalf("expected no mutation results, got %v", got.results)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("invalid options blocked instead of returning an error")
				}
			})
		}
	}
}
