package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/sivchari/gomu/pkg/gomu"
	"github.com/spf13/cobra"
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

func TestSplitPathAndTestArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantPath     string
		wantTestArgs []string
	}{
		{
			name:         "no path no dash",
			args:         []string{"run"},
			wantPath:     ".",
			wantTestArgs: nil,
		},
		{
			name:         "path no dash",
			args:         []string{"run", "./pkg"},
			wantPath:     "./pkg",
			wantTestArgs: nil,
		},
		{
			name:         "dash without path",
			args:         []string{"run", "--", "-short"},
			wantPath:     ".",
			wantTestArgs: []string{"-short"},
		},
		{
			name:         "path and dash",
			args:         []string{"run", "./pkg", "--", "-short"},
			wantPath:     "./pkg",
			wantTestArgs: []string{"-short"},
		},
		{
			name:         "multiple flags after dash preserve boundaries",
			args:         []string{"run", "./pkg", "--", "-short", "-race", "-run", "^TestUnit"},
			wantPath:     "./pkg",
			wantTestArgs: []string{"-short", "-race", "-run", "^TestUnit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "run",
				Args: validateRunArgs,
				RunE: func(_ *cobra.Command, _ []string) error { return nil },
			}
			cmd.Flags().BoolP("list", "l", false, "")

			cmd.SetArgs(tt.args[1:])

			var gotPath string

			var gotTestArgs []string

			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				gotPath, gotTestArgs = splitPathAndTestArgs(cmd, args)

				return nil
			}
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}

			if !reflect.DeepEqual(gotTestArgs, tt.wantTestArgs) {
				t.Errorf("testArgs = %v, want %v", gotTestArgs, tt.wantTestArgs)
			}
		})
	}
}

func TestRunArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{"run"},
			wantErr: false,
		},
		{
			name:    "one path arg",
			args:    []string{"run", "./pkg"},
			wantErr: false,
		},
		{
			name:    "two path args without dash still errors",
			args:    []string{"run", "a", "b"},
			wantErr: true,
		},
		{
			name:    "dash with no path is allowed",
			args:    []string{"run", "--", "-short"},
			wantErr: false,
		},
		{
			name:    "path and dash is allowed",
			args:    []string{"run", "./pkg", "--", "-short"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// --list must precede "--" so it still parses as a flag rather
			// than being swallowed into the forwarded test args; the run
			// path never actually executes because --list short-circuits it.
			argsWithList := append([]string{"run", "--list"}, tt.args[1:]...)

			rootCmd.SetArgs(argsWithList)
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)

			err := rootCmd.Execute()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Reset the list flag's value (not just Changed), since --list
			// was set explicitly above and would otherwise leak into later
			// tests that don't pass --list.
			if err := runCmd.Flags().Set("list", "false"); err != nil {
				t.Fatalf("reset list flag: %v", err)
			}

			runCmd.Flags().Visit(func(f *pflag.Flag) {
				f.Changed = false
			})
		})
	}
}

func TestRunRejectsConflictingTestArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	rootCmd.SetArgs([]string{"run", "--", "-overlay=foo.json"})
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an actionable error rejecting -overlay, got none")
	}

	if !strings.Contains(err.Error(), "-overlay=foo.json") {
		t.Errorf("expected error to name the offending flag, got: %v", err)
	}
}

func TestRunListWarnsIgnoredFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantWarning bool
		wantFlag    string
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
			wantFlag:    "--output",
		},
		{
			name:        "list with include-generated warns about ignored flag",
			args:        []string{"run", "--list", "--include-generated"},
			wantWarning: true,
			wantFlag:    "--include-generated",
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
				if !strings.Contains(errOut, tt.wantFlag) {
					t.Errorf("expected warning mentioning %s, got: %q", tt.wantFlag, errOut)
				}
			} else if errOut != "" {
				t.Errorf("expected no warning, got: %q", errOut)
			}

			// Reset flags for subsequent test cases since runCmd is a
			// package-level var shared across tests.
			if err := runCmd.Flags().Set("output", "console"); err != nil {
				t.Fatalf("reset output flag: %v", err)
			}

			if err := runCmd.Flags().Set("include-generated", "false"); err != nil {
				t.Fatalf("reset include-generated flag: %v", err)
			}

			runCmd.Flags().Visit(func(f *pflag.Flag) {
				f.Changed = false
			})
		})
	}
}
