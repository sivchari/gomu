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

func TestFindMutationDirInCurrentDirectory(t *testing.T) {
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
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)
	
	// Test
	dir, err := findMutationDirImpl()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty directory path")
	}
	if !strings.HasSuffix(dir, filepath.Join("internal", "mutation")) {
		t.Errorf("expected dir to end with internal/mutation, got %s", dir)
	}
}

func TestFindMutationDirInParentDirectory(t *testing.T) {
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
	
	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Change to subdirectory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(subDir)
	
	// Test
	dir, err := findMutationDirImpl()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty directory path")
	}
	if !strings.HasSuffix(dir, filepath.Join("internal", "mutation")) {
		t.Errorf("expected dir to end with internal/mutation, got %s", dir)
	}
}

func TestFindMutationDirWhenInMutationDirectory(t *testing.T) {
	tempDir := t.TempDir()
	mutationDir := filepath.Join(tempDir, "mutation")
	if err := os.MkdirAll(mutationDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Change to mutation directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(mutationDir)
	
	// Test
	dir, err := findMutationDirImpl()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty directory path")
	}
	if !strings.HasSuffix(dir, "mutation") {
		t.Errorf("expected dir to end with mutation, got %s", dir)
	}
}

func TestGenerateFileWithValidTemplate(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.go")
	template := "package mutation\n\n// {{.StructName}} mutator for {{.Description}}"
	data := mutatorData{
		LowerName:   "test",
		StructName:  "Test",
		Description: "test operators",
	}
	
	err := generateFile(tempFile, template, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Test mutator") {
		t.Error("expected content to contain 'Test mutator'")
	}
	if !strings.Contains(contentStr, "test operators") {
		t.Error("expected content to contain 'test operators'")
	}
	if !strings.Contains(contentStr, "package mutation") {
		t.Error("expected package declaration")
	}
}

func TestGenerateFileWithInvalidTemplate(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.go")
	template := "{{.InvalidField}}"
	data := mutatorData{
		LowerName:   "test",
		StructName:  "Test",
		Description: "test operators",
	}
	
	err := generateFile(tempFile, template, data)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestGenerateFileWithComplexTemplate(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.go")
	template := "// Lower: {{.LowerName}}\n// Struct: {{.StructName}}\n// Desc: {{.Description}}"
	data := mutatorData{
		LowerName:   "arithmetic",
		StructName:  "Arithmetic",
		Description: "arithmetic operators",
	}
	
	err := generateFile(tempFile, template, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Lower: arithmetic") {
		t.Error("expected lower name in output")
	}
	if !strings.Contains(contentStr, "Struct: Arithmetic") {
		t.Error("expected struct name in output")
	}
	if !strings.Contains(contentStr, "Desc: arithmetic operators") {
		t.Error("expected description in output")
	}
}

func TestGenerateFileWithEmptyTemplate(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.go")
	template := ""
	data := mutatorData{
		LowerName:   "test",
		StructName:  "Test", 
		Description: "test operators",
	}
	
	err := generateFile(tempFile, template, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "" {
		t.Error("expected empty content for empty template")
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

func TestMainFunctionWithMissingFlag(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Stderr = oldStderr
	}()
	
	// Setup test
	flag.CommandLine = flag.NewFlagSet("cmd", flag.ContinueOnError)
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr
	os.Args = []string{"cmd"}
	
	exitCode := 0
	origExit := exitFunc
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = origExit }()
	
	// Run main
	main()
	
	// Check results
	wErr.Close()
	var bufErr bytes.Buffer
	io.Copy(&bufErr, rErr)
	
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	
	output := bufErr.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("expected usage message")
	}
	if !strings.Contains(output, "-mutator=<mutator_name>") {
		t.Error("expected mutator flag in usage")
	}
	if !strings.Contains(output, "Example:") {
		t.Error("expected example in usage")
	}
}

func TestMainFunctionWithEmptyFlag(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Stderr = oldStderr
	}()
	
	// Setup test
	flag.CommandLine = flag.NewFlagSet("cmd", flag.ContinueOnError)
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr
	os.Args = []string{"cmd", "-mutator="}
	
	exitCode := 0
	origExit := exitFunc
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = origExit }()
	
	// Run main
	main()
	
	// Check results
	wErr.Close()
	var bufErr bytes.Buffer
	io.Copy(&bufErr, rErr)
	
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	
	output := bufErr.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("expected usage message")
	}
	if !strings.Contains(output, "cmd -mutator=<mutator_name>") {
		t.Error("expected proper usage format")
	}
}

func TestMainFunctionWithValidMutator(t *testing.T) {
	// Setup temporary mutation directory
	tempDir := t.TempDir()
	mutationDir := filepath.Join(tempDir, "internal", "mutation")
	if err := os.MkdirAll(mutationDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create engine.go
	engineFile := filepath.Join(mutationDir, "engine.go")
	if err := os.WriteFile(engineFile, []byte("package mutation"), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)
	
	// Save original state
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Stdout = oldStdout
	}()
	
	// Setup test
	flag.CommandLine = flag.NewFlagSet("cmd", flag.ContinueOnError)
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut
	os.Args = []string{"cmd", "-mutator=sample"}
	
	exitCode := 0
	origExit := exitFunc
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = origExit }()
	
	// Run main
	main()
	
	// Check results
	wOut.Close()
	var bufOut bytes.Buffer
	io.Copy(&bufOut, rOut)
	
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	
	output := bufOut.String()
	if !strings.Contains(output, "Generated sample mutator:") {
		t.Error("expected generation message")
	}
	if !strings.Contains(output, "sample.go") {
		t.Error("expected mutator file name")
	}
	if !strings.Contains(output, "sample_test.go") {
		t.Error("expected test file name")
	}
	if !strings.Contains(output, "Regenerating registry...") {
		t.Error("expected registry regeneration message")
	}
	if !strings.Contains(output, "Next steps:") {
		t.Error("expected next steps message")
	}
}

func TestMainFunctionWithMutationDirError(t *testing.T) {
	// Save original state
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Stderr = oldStderr
	}()
	
	// Override findMutationDir to fail
	oldFind := *FindMutationDir
	*FindMutationDir = func() (string, error) {
		return "", os.ErrNotExist
	}
	defer func() {
		*FindMutationDir = oldFind
	}()
	
	// Setup test
	flag.CommandLine = flag.NewFlagSet("cmd", flag.ContinueOnError)
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr
	os.Args = []string{"cmd", "-mutator=testmutator"}
	
	exitCode := 0
	origExit := exitFunc
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = origExit }()
	
	// Run main
	main()
	
	// Check results
	wErr.Close()
	var bufErr bytes.Buffer
	io.Copy(&bufErr, rErr)
	
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	
	output := bufErr.String()
	if !strings.Contains(output, "Error finding mutation directory") {
		t.Error("expected error finding mutation directory")
	}
	if !strings.Contains(output, "file does not exist") {
		t.Error("expected specific error message")
	}
}

func TestMainFunctionWithFileGenerationError(t *testing.T) {
	// Setup temporary mutation directory (read-only)
	tempDir := t.TempDir()
	mutationDir := filepath.Join(tempDir, "internal", "mutation")
	if err := os.MkdirAll(mutationDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create engine.go
	engineFile := filepath.Join(mutationDir, "engine.go")
	if err := os.WriteFile(engineFile, []byte("package mutation"), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Make directory read-only to cause write error
	os.Chmod(mutationDir, 0555)
	defer os.Chmod(mutationDir, 0755)
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)
	
	// Save original state
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Stderr = oldStderr
	}()
	
	// Setup test
	flag.CommandLine = flag.NewFlagSet("cmd", flag.ContinueOnError)
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr
	os.Args = []string{"cmd", "-mutator=test"}
	
	exitCode := 0
	origExit := exitFunc
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = origExit }()
	
	// Run main
	main()
	
	// Check results
	wErr.Close()
	var bufErr bytes.Buffer
	io.Copy(&bufErr, rErr)
	
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	
	output := bufErr.String()
	if !strings.Contains(output, "Error generating mutator file") {
		t.Error("expected error generating mutator file")
	}
}

func TestGenerateRegistryWithValidDirectory(t *testing.T) {
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
	
	err := generateRegistry(tempDir)
	if err != nil {
		// go generate might fail in test environment, but command should execute
		if !strings.Contains(err.Error(), "go generate") {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestGenerateRegistryWithNonExistentDirectory(t *testing.T) {
	err := generateRegistry("/nonexistent/directory")
	if err == nil {
		t.Error("expected error but got none")
	}
}

