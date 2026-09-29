package gomu

import (
	"strings"
	"testing"
)

func TestValidateTestArgs(t *testing.T) {
	tests := []struct {
		name          string
		testArgs      []string
		wantErr       bool
		errorContains string
	}{
		{
			name:     "nil test args",
			testArgs: nil,
			wantErr:  false,
		},
		{
			name:     "accepts -short",
			testArgs: []string{"-short"},
			wantErr:  false,
		},
		{
			name:     "accepts -race",
			testArgs: []string{"-race"},
			wantErr:  false,
		},
		{
			name:     "accepts -run with separate value",
			testArgs: []string{"-run", "^TestUnit"},
			wantErr:  false,
		},
		{
			name:     "accepts -exec",
			testArgs: []string{"-exec", "foo"},
			wantErr:  false,
		},
		{
			name:          "rejects -overlay with equals",
			testArgs:      []string{"-overlay=foo.json"},
			wantErr:       true,
			errorContains: "-overlay=foo.json",
		},
		{
			name:          "rejects --overlay double dash",
			testArgs:      []string{"--overlay=foo.json"},
			wantErr:       true,
			errorContains: "--overlay=foo.json",
		},
		{
			name:          "rejects -c",
			testArgs:      []string{"-c"},
			wantErr:       true,
			errorContains: "-c",
		},
		{
			name:          "rejects -o",
			testArgs:      []string{"-o", "bin"},
			wantErr:       true,
			errorContains: "-o",
		},
		{
			name:          "rejects -args",
			testArgs:      []string{"-args", "x"},
			wantErr:       true,
			errorContains: "-args",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTestArgs(tt.testArgs)
			if tt.wantErr && err == nil {
				t.Fatal("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error to contain %q, got: %v", tt.errorContains, err)
			}
		})
	}
}
