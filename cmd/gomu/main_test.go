package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sivchari/gomu/pkg/gomu"
	"github.com/spf13/pflag"
)

func TestListMutators(t *testing.T) {
	var buf bytes.Buffer
	if err := listMutators(&buf); err != nil {
		t.Fatalf("listMutators: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Supported mutators") {
		t.Errorf("output missing header:\n%s", out)
	}

	mutators := gomu.SupportedMutators()
	if len(mutators) == 0 {
		t.Fatal("no supported mutators reported")
	}

	for _, m := range mutators {
		if !strings.Contains(out, m.Name) {
			t.Errorf("output missing mutator name %q:\n%s", m.Name, out)
		}

		if !strings.Contains(out, m.Description) {
			t.Errorf("output missing description %q:\n%s", m.Description, out)
		}
	}
}

func TestRunListWarnsIgnoredFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantWarning bool
	}{
		{
			name:        "list alone produces no warning",
			args:        []string{"run", "--list"},
			wantWarning: false,
		},
		{
			name:        "list with output warns about ignored flag",
			args:        []string{"run", "--list", "--output", "json"},
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			rootCmd.SetArgs(tt.args)
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			errOut := stderr.String()

			if tt.wantWarning {
				if !strings.Contains(errOut, "--output") {
					t.Errorf("expected warning mentioning --output, got: %q", errOut)
				}
			} else if errOut != "" {
				t.Errorf("expected no warning, got: %q", errOut)
			}

			// Reset flags for subsequent test cases since runCmd is a
			// package-level var shared across tests.
			if err := runCmd.Flags().Set("output", "console"); err != nil {
				t.Fatalf("reset output flag: %v", err)
			}

			runCmd.Flags().Visit(func(f *pflag.Flag) {
				f.Changed = false
			})
		})
	}
}
