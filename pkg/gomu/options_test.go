package gomu

import (
	"context"
	"strings"
	"testing"
)

func TestRunRejectsInvalidExecutionOptions(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		timeout int
		wantErr string
	}{
		{name: "zero workers", workers: 0, timeout: 5, wantErr: "workers must be at least 1 (got 0)"},
		{name: "negative workers", workers: -1, timeout: 5, wantErr: "workers must be at least 1 (got -1)"},
		{name: "zero timeout", workers: 1, timeout: 0, wantErr: "timeout must be at least 1 second (got 0)"},
		{name: "negative timeout", workers: 1, timeout: -1, wantErr: "timeout must be at least 1 second (got -1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A bare engine proves validation happens before discovery or execution.
			engine := &Engine{}
			opts := &RunOptions{Workers: tt.workers, Timeout: tt.timeout, CIMode: true}

			err := engine.Run(context.Background(), ".", opts)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
