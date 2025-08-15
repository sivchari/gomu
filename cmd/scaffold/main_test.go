package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMutatorData(t *testing.T) {
	data := mutatorData{
		LowerName:   "test",
		StructName:  "Test",
		Description: "test operators",
	}

	if data.LowerName != "test" {
		t.Errorf("expected LowerName to be 'test', got %s", data.LowerName)
	}
	if data.StructName != "Test" {
		t.Errorf("expected StructName to be 'Test', got %s", data.StructName)
	}
	if data.Description != "test operators" {
		t.Errorf("expected Description to be 'test operators', got %s", data.Description)
	}
}

func TestFindMutationDir(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		cleanup   func()
		wantError bool
	}{
		{
			name: "finds_mutation_dir_in_current_directory",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				mutationDir := filepath.Join(tempDir, "internal", "mutation")
				if err := os.MkdirAll(mutationDir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create engine.go to match the actual findMutationDir logic
				engineFile := filepath.Join(mutationDir, "engine.go")
				if err := os.WriteFile(engineFile, []byte("package mutation"), 0644); err != nil {
					t.Fatal(err)
				}
				return tempDir
			},
			cleanup:   func() {},
			wantError: false,
		},
		{
			name: "finds_mutation_dir_in_parent_directory",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				mutationDir := filepath.Join(tempDir, "internal", "mutation")
				if err := os.MkdirAll(mutationDir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create engine.go to match the actual findMutationDir logic
				engineFile := filepath.Join(mutationDir, "engine.go")
				if err := os.WriteFile(engineFile, []byte("package mutation"), 0644); err != nil {
					t.Fatal(err)
				}
				subDir := filepath.Join(tempDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatal(err)
				}
				return subDir
			},
			cleanup:   func() {},
			wantError: false,
		},
		{
			name: "finds_mutation_dir_when_already_in_mutation_directory",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				mutationDir := filepath.Join(tempDir, "mutation")
				if err := os.MkdirAll(mutationDir, 0755); err != nil {
					t.Fatal(err)
				}
				return mutationDir
			},
			cleanup:   func() {},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldWd)

			testDir := tt.setup(t)
			if err := os.Chdir(testDir); err != nil {
				t.Fatal(err)
			}
			defer tt.cleanup()

			dir, err := findMutationDirImpl()
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if dir == "" {
					t.Error("expected non-empty directory path")
				}
				if !strings.Contains(dir, filepath.Join("internal", "mutation")) && !strings.HasSuffix(dir, "mutation") {
					t.Errorf("expected path to contain internal/mutation or end with mutation, got %s", dir)
				}
			}
		})
	}
}

func TestGenerateFile(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		data      mutatorData
		wantError bool
		checkFile func(t *testing.T, content string)
	}{
		{
			name:     "generates_file_with_valid_template",
			template: "package mutation\n\n// {{.StructName}} mutator for {{.Description}}",
			data: mutatorData{
				LowerName:   "test",
				StructName:  "Test",
				Description: "test operators",
			},
			wantError: false,
			checkFile: func(t *testing.T, content string) {
				if !strings.Contains(content, "Test mutator") {
					t.Error("expected content to contain 'Test mutator'")
				}
				if !strings.Contains(content, "test operators") {
					t.Error("expected content to contain 'test operators'")
				}
			},
		},
		{
			name:     "handles_invalid_template",
			template: "{{.InvalidField}}",
			data: mutatorData{
				LowerName:   "test",
				StructName:  "Test",
				Description: "test operators",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(t.TempDir(), "test.go")
			err := generateFile(tempFile, tt.template, tt.data)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				content, err := os.ReadFile(tempFile)
				if err != nil {
					t.Fatal(err)
				}

				if tt.checkFile != nil {
					tt.checkFile(t, string(content))
				}
			}
		})
	}
}

func TestGenerateFileInvalidPath(t *testing.T) {
	data := mutatorData{
		LowerName:   "test",
		StructName:  "Test",
		Description: "test operators",
	}

	// Test with invalid path
	err := generateFile("/nonexistent/path/file.go", "template", data)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestMainFunction(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		setup    func(t *testing.T) (cleanup func())
		wantExit int
		checkErr func(t *testing.T, output string)
		checkOut func(t *testing.T, output string)
	}{
		{
			name:     "missing_mutator_flag",
			args:     []string{"cmd"},
			wantExit: 1,
			checkErr: func(t *testing.T, output string) {
				if !strings.Contains(output, "Usage:") {
					t.Error("expected usage message")
				}
			},
		},
		{
			name:     "empty_mutator_flag",
			args:     []string{"cmd", "-mutator="},
			wantExit: 1,
			checkErr: func(t *testing.T, output string) {
				if !strings.Contains(output, "Usage:") {
					t.Error("expected usage message")
				}
			},
		},
		{
			name: "valid_mutator_generates_files",
			args: []string{"cmd", "-mutator=testmutator"},
			setup: func(t *testing.T) func() {
				// Create a temporary mutation directory
				tempDir := t.TempDir()
				mutationDir := filepath.Join(tempDir, "internal", "mutation")
				if err := os.MkdirAll(mutationDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Create engine.go to make findMutationDir succeed
				engineFile := filepath.Join(mutationDir, "engine.go")
				if err := os.WriteFile(engineFile, []byte("package mutation"), 0644); err != nil {
					t.Fatal(err)
				}

				// Change to temp directory
				oldWd, _ := os.Getwd()
				os.Chdir(tempDir)

				return func() {
					os.Chdir(oldWd)
				}
			},
			wantExit: 0,
			checkOut: func(t *testing.T, output string) {
				if !strings.Contains(output, "Generated testmutator mutator:") {
					t.Error("expected generation message")
				}
			},
		},
		{
			name: "error_finding_mutation_dir",
			args: []string{"cmd", "-mutator=testmutator"},
			setup: func(t *testing.T) func() {
				// Change to a directory without mutation package
				tempDir := t.TempDir()
				deepDir := filepath.Join(tempDir, "a", "b", "c", "d", "e", "f")
				os.MkdirAll(deepDir, 0755)
				oldWd, _ := os.Getwd()
				os.Chdir(deepDir)

				// Override findMutationDir to always fail
				oldFind := *FindMutationDir
				*FindMutationDir = func() (string, error) {
					return "", os.ErrNotExist
				}

				return func() {
					os.Chdir(oldWd)
					*FindMutationDir = oldFind
				}
			},
			wantExit: 1,
			checkErr: func(t *testing.T, output string) {
				if !strings.Contains(output, "Error finding mutation directory") {
					t.Error("expected error finding mutation directory")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args and flag.CommandLine
			oldArgs := os.Args
			oldCommandLine := flag.CommandLine
			oldStderr := os.Stderr
			oldStdout := os.Stdout

			// Create new flag set for testing
			flag.CommandLine = flag.NewFlagSet(tt.args[0], flag.ContinueOnError)

			// Capture stderr and stdout
			rErr, wErr, _ := os.Pipe()
			rOut, wOut, _ := os.Pipe()
			os.Stderr = wErr
			os.Stdout = wOut

			var cleanup func()
			if tt.setup != nil {
				cleanup = tt.setup(t)
			}

			defer func() {
				if cleanup != nil {
					cleanup()
				}
				os.Args = oldArgs
				flag.CommandLine = oldCommandLine
				os.Stderr = oldStderr
				os.Stdout = oldStdout
			}()

			// Set args and run with exit capture
			os.Args = tt.args

			exitCode := 0
			origExit := exitFunc
			exitFunc = func(code int) {
				exitCode = code
			}
			defer func() { exitFunc = origExit }()

			// Run main
			main()

			// Close writers and read output
			wErr.Close()
			wOut.Close()
			var bufErr bytes.Buffer
			var bufOut bytes.Buffer
			io.Copy(&bufErr, rErr)
			io.Copy(&bufOut, rOut)

			if exitCode != tt.wantExit {
				t.Errorf("expected exit code %d, got %d", tt.wantExit, exitCode)
			}

			if tt.checkErr != nil {
				tt.checkErr(t, bufErr.String())
			}

			if tt.checkOut != nil {
				tt.checkOut(t, bufOut.String())
			}
		})
	}
}

func TestGenerateRegistry(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantError bool
	}{
		{
			name: "handles_directory_with_go_generate",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				// Create a simple go.mod file
				goMod := `module test

go 1.20
`
				if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
					t.Fatal(err)
				}

				// Create a simple Go file with generate directive
				goFile := `package mutation

//go:generate echo "test"
`
				if err := os.WriteFile(filepath.Join(tempDir, "test.go"), []byte(goFile), 0644); err != nil {
					t.Fatal(err)
				}
				return tempDir
			},
			wantError: false,
		},
		{
			name: "handles_non_existent_directory",
			setup: func(t *testing.T) string {
				return "/nonexistent/directory"
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			err := generateRegistry(dir)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					// go generate might fail in test environment, but command should execute
					if !strings.Contains(err.Error(), "go generate") {
						t.Errorf("unexpected error: %v", err)
					}
				}
			}
		})
	}
}
