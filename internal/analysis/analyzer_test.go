package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "creates analyzer successfully",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := New()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if analyzer == nil {
				t.Fatal("analyzer should not be nil")
			}

			if analyzer.fileSet == nil {
				t.Error("fileSet should not be nil")
			}

			if analyzer.typeInfo == nil {
				t.Error("typeInfo should not be nil")
			} else {
				// Verify typeInfo is properly initialized
				if analyzer.typeInfo.Types == nil {
					t.Error("Types map should be initialized")
				}

				if analyzer.typeInfo.Uses == nil {
					t.Error("Uses map should be initialized")
				}

				if analyzer.typeInfo.Defs == nil {
					t.Error("Defs map should be initialized")
				}
			}
		})
	}
}

func TestFindTargetFiles(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(t *testing.T) string
		expectError bool
		expectCount int
		expectFiles []string
	}{
		{
			name: "finds go files in simple directory",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				// Create some Go files
				os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "util.go"), []byte("package main"), 0644)

				// Create a test file (should be excluded)
				os.WriteFile(filepath.Join(tempDir, "main_test.go"), []byte("package main"), 0644)

				// Create a non-Go file (should be excluded)
				os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# README"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 2,
			expectFiles: []string{"main.go", "util.go"},
		},
		{
			name: "excludes vendor directory",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

				// Create vendor directory with Go files
				vendorDir := filepath.Join(tempDir, "vendor", "github.com", "pkg")
				os.MkdirAll(vendorDir, 0755)
				os.WriteFile(filepath.Join(vendorDir, "vendor.go"), []byte("package pkg"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 1,
			expectFiles: []string{"main.go"},
		},
		{
			name: "excludes testdata directory",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				os.WriteFile(filepath.Join(tempDir, "analyzer.go"), []byte("package main"), 0644)

				// Create testdata directory with Go files
				testdataDir := filepath.Join(tempDir, "testdata")
				os.MkdirAll(testdataDir, 0755)
				os.WriteFile(filepath.Join(testdataDir, "test.go"), []byte("package test"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 1,
			expectFiles: []string{"analyzer.go"},
		},
		{
			name: "handles nested directories",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				// Create nested structure
				os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)

				subDir := filepath.Join(tempDir, "pkg", "utils")
				os.MkdirAll(subDir, 0755)
				os.WriteFile(filepath.Join(subDir, "helper.go"), []byte("package utils"), 0644)
				os.WriteFile(filepath.Join(subDir, "helper_test.go"), []byte("package utils"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 2,
			expectFiles: []string{"main.go", "helper.go"},
		},
		{
			name: "handles empty directory",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: false,
			expectCount: 0,
			expectFiles: []string{},
		},
		{
			name: "handles non-existent directory",
			setupFiles: func(_ *testing.T) string {
				return "/non/existent/path"
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "handles directory with only test files",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()
				os.WriteFile(filepath.Join(tempDir, "foo_test.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "bar_test.go"), []byte("package main"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 0,
			expectFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootPath := tt.setupFiles(t)

			analyzer, err := New()
			if err != nil {
				t.Fatalf("failed to create analyzer: %v", err)
			}

			files, err := analyzer.FindTargetFiles(rootPath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(files) != tt.expectCount {
				t.Errorf("expected %d files, got %d", tt.expectCount, len(files))
			}

			// Verify expected files are found
			for _, expectedFile := range tt.expectFiles {
				found := false

				for _, file := range files {
					if strings.HasSuffix(file, expectedFile) {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("expected to find file %s", expectedFile)
				}
			}
		})
	}
}

func TestFindChangedFiles(t *testing.T) {
	tests := []struct {
		name        string
		allFiles    []string
		setupGit    func(t *testing.T) func()
		expectError bool
		expectFiles []string
	}{
		{
			name: "finds changed files from git diff",
			allFiles: []string{
				"/project/main.go",
				"/project/util.go",
				"/project/helper.go",
			},
			setupGit: func(t *testing.T) func() {
				// Skip if not in a git repo
				if _, err := os.Stat(".git"); os.IsNotExist(err) {
					t.Skip("Not in a git repository")
				}

				return func() {}
			},
			expectError: false,
			// The actual result depends on git state
		},
		{
			name:     "handles empty file list",
			allFiles: []string{},
			setupGit: func(_ *testing.T) func() {
				return func() {}
			},
			expectError: false,
			expectFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupGit(t)
			defer cleanup()

			analyzer, err := New()
			if err != nil {
				t.Fatalf("failed to create analyzer: %v", err)
			}

			files, err := analyzer.FindChangedFiles(tt.allFiles)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			// Don't fail on git errors in test environment
			if err != nil && strings.Contains(err.Error(), "git diff") {
				t.Skip("Git not available or not in a repository")
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify result is a subset of input files
			for _, file := range files {
				found := false

				for _, allFile := range tt.allFiles {
					if file == allFile {
						found = true

						break
					}
				}

				if !found {
					t.Errorf("returned file %s not in input files", file)
				}
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, info *FileInfo)
	}{
		{
			name: "parses valid Go file",
			setupFile: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "test.go")
				content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
				os.WriteFile(filePath, []byte(content), 0644)

				return filePath
			},
			expectError: false,
			validate: func(t *testing.T, info *FileInfo) {
				if info.FileAST == nil {
					t.Error("FileAST should not be nil")
				}
				if info.FileAST.Name.Name != "main" {
					t.Errorf("expected package name 'main', got %s", info.FileAST.Name.Name)
				}
				if info.Hash == "" {
					t.Error("Hash should not be empty")
				}
				if len(info.FileAST.Decls) != 2 { // import and func
					t.Errorf("expected 2 declarations, got %d", len(info.FileAST.Decls))
				}
			},
		},
		{
			name: "handles syntax error",
			setupFile: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "invalid.go")
				content := `package main

func main() {
	fmt.Println("Missing import!"
}
`
				os.WriteFile(filePath, []byte(content), 0644)

				return filePath
			},
			expectError: true,
		},
		{
			name: "handles non-existent file",
			setupFile: func(_ *testing.T) string {
				return "/non/existent/file.go"
			},
			expectError: true,
		},
		{
			name: "parses file with multiple functions",
			setupFile: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "multi.go")
				content := `package utils

// Add adds two numbers
func Add(a, b int) int {
	return a + b
}

// Subtract subtracts b from a
func Subtract(a, b int) int {
	return a - b
}
`
				os.WriteFile(filePath, []byte(content), 0644)

				return filePath
			},
			expectError: false,
			validate: func(t *testing.T, info *FileInfo) {
				if info.FileAST.Name.Name != "utils" {
					t.Errorf("expected package name 'utils', got %s", info.FileAST.Name.Name)
				}

				funcCount := 0
				for _, decl := range info.FileAST.Decls {
					if _, ok := decl.(*ast.FuncDecl); ok {
						funcCount++
					}
				}
				if funcCount != 2 {
					t.Errorf("expected 2 functions, got %d", funcCount)
				}
			},
		},
		{
			name: "handles empty file",
			setupFile: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "empty.go")
				os.WriteFile(filePath, []byte(""), 0644)

				return filePath
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFile(t)

			analyzer, err := New()
			if err != nil {
				t.Fatalf("failed to create analyzer: %v", err)
			}

			info, err := analyzer.ParseFile(filePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if info == nil {
				t.Fatal("info should not be nil")
			}

			if info.Path != filePath {
				t.Errorf("expected path %s, got %s", filePath, info.Path)
			}

			if tt.validate != nil {
				tt.validate(t, info)
			}
		})
	}
}

func TestGetPosition(t *testing.T) {
	analyzer, err := New()
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	// Create a simple AST to test position
	src := `package main

func test() {
	x := 1
}`

	file, err := parser.ParseFile(analyzer.fileSet, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test file: %v", err)
	}

	// Get position of the function declaration
	funcDecl, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("expected function declaration")
	}

	pos := analyzer.GetPosition(funcDecl.Pos())

	if pos.Filename != "test.go" {
		t.Errorf("expected filename 'test.go', got %s", pos.Filename)
	}

	if pos.Line != 3 {
		t.Errorf("expected line 3, got %d", pos.Line)
	}

	if pos.Column != 1 {
		t.Errorf("expected column 1, got %d", pos.Column)
	}
}

func TestGetFileSet(t *testing.T) {
	analyzer, err := New()
	if err != nil {
		t.Fatalf("failed to create analyzer: %v", err)
	}

	fileSet := analyzer.GetFileSet()

	if fileSet == nil {
		t.Error("fileSet should not be nil")
	}

	// Verify it's the same instance
	if fileSet != analyzer.fileSet {
		t.Error("GetFileSet should return the same fileSet instance")
	}
}

func TestCalculateFileHash(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: "0",
		},
		{
			name:     "simple content",
			content:  []byte("hello"),
			expected: "5",
		},
		{
			name:     "longer content",
			content:  []byte("package main\n\nfunc main() {}\n"),
			expected: fmt.Sprintf("%x", 29), // Actual length is 29
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := calculateFileHash(tt.content)

			if hash != tt.expected {
				t.Errorf("expected hash %s, got %s", tt.expected, hash)
			}
		})
	}
}

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected bool
	}{
		{"valid main", "main", true},
		{"valid master", "master", true},
		{"valid with slash", "feature/new-feature", true},
		{"valid with dots", "release-1.2.3", true},
		{"valid with underscore", "bug_fix_123", true},
		{"valid complex", "users/john/feature-123", true},
		{"invalid starts with hyphen", "-main", false},
		{"invalid empty", "", false},
		{"invalid with spaces", "my branch", false},
		{"invalid with semicolon", "main;echo hack", false},
		{"invalid with pipe", "main|cat", false},
		{"invalid with backtick", "main`pwd`", false},
		{"invalid with dollar", "main$USER", false},
		{"invalid with ampersand", "main&", false},
		{"invalid with parentheses", "main()", false},
		{"invalid with quotes", "main'", false},
		{"invalid with double quotes", "main\"", false},
		{"valid long name", "feature/very-long-branch-name-with-many-words-123", true},
		{"valid numbers only after first char", "1.0.0", true},
		{"invalid starts with dot", ".hidden", false},
		{"invalid starts with slash", "/main", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBranchName(tt.branch)

			if result != tt.expected {
				t.Errorf("isValidBranchName(%q) = %v, expected %v", tt.branch, result, tt.expected)
			}
		})
	}
}

func TestGetTypeInfo(t *testing.T) {
	tests := []struct {
		name      string
		setupFile func(t *testing.T) (string, *ast.File)
		expectNil bool
	}{
		{
			name: "gets type info for valid file",
			setupFile: func(t *testing.T) (string, *ast.File) {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "main.go")
				content := `package main

func Add(a, b int) int {
	return a + b
}
`
				os.WriteFile(filePath, []byte(content), 0644)

				fset := token.NewFileSet()
				file, _ := parser.ParseFile(fset, filePath, content, parser.ParseComments)

				return filePath, file
			},
			expectNil: false,
		},
		{
			name: "handles file with syntax errors gracefully",
			setupFile: func(t *testing.T) (string, *ast.File) {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "broken.go")

				// Create a file that parses but has type errors
				content := `package main

func Broken() {
	undefinedFunc()
}
`
				os.WriteFile(filePath, []byte(content), 0644)

				fset := token.NewFileSet()
				file, _ := parser.ParseFile(fset, filePath, content, parser.ParseComments)

				return filePath, file
			},
			expectNil: true, // Type checking will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := New()
			if err != nil {
				t.Fatalf("failed to create analyzer: %v", err)
			}

			filePath, fileAST := tt.setupFile(t)

			typeInfo := analyzer.getTypeInfo(fileAST, filePath)

			if tt.expectNil {
				if typeInfo != nil {
					t.Error("expected nil typeInfo")
				}
			}
		})
	}
}

func TestParsePackageFiles(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(t *testing.T) string
		expectError bool
		expectCount int
	}{
		{
			name: "parses all non-test files in package",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				// Create several Go files
				os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "util.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "main_test.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# README"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 2, // main.go and util.go
		},
		{
			name: "handles directory with only test files",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				os.WriteFile(filepath.Join(tempDir, "foo_test.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "bar_test.go"), []byte("package main"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "handles empty directory",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "skips files with parse errors",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()

				// Valid file
				os.WriteFile(filepath.Join(tempDir, "valid.go"), []byte("package main"), 0644)

				// Invalid file
				os.WriteFile(filepath.Join(tempDir, "invalid.go"), []byte("not valid go code!"), 0644)

				return tempDir
			},
			expectError: false,
			expectCount: 1, // Only valid.go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgDir := tt.setupFiles(t)

			analyzer, err := New()
			if err != nil {
				t.Fatalf("failed to create analyzer: %v", err)
			}

			files, err := analyzer.parsePackageFiles(pkgDir)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(files) != tt.expectCount {
				t.Errorf("expected %d files, got %d", tt.expectCount, len(files))
			}
		})
	}
}
