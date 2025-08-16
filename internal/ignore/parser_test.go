package ignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	// Test: Create a new parser
	parser := New()
	if parser == nil {
		t.Fatal("parser is nil")
	}

	if len(parser.patterns) != 0 {
		t.Errorf("initial pattern count is wrong: expected=0, actual=%d", len(parser.patterns))
	}
}

func TestLoadFromReader(t *testing.T) {
	// Test: Load patterns from reader
	testCases := []struct {
		name     string
		content  string
		expected []Pattern
	}{
		{
			name:     "empty content",
			content:  "",
			expected: []Pattern{},
		},
		{
			name: "basic patterns",
			content: `*.go
vendor/
testdata/`,
			expected: []Pattern{
				{Pattern: "*.go", Negate: false},
				{Pattern: "vendor/", Negate: false},
				{Pattern: "testdata/", Negate: false},
			},
		},
		{
			name: "comments and empty lines",
			content: `# This is a comment
*.go

# Another comment
vendor/`,
			expected: []Pattern{
				{Pattern: "*.go", Negate: false},
				{Pattern: "vendor/", Negate: false},
			},
		},
		{
			name: "negation patterns",
			content: `*.go
!important.go
vendor/
!vendor/keep/`,
			expected: []Pattern{
				{Pattern: "*.go", Negate: false},
				{Pattern: "important.go", Negate: true},
				{Pattern: "vendor/", Negate: false},
				{Pattern: "vendor/keep/", Negate: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := New()
			reader := strings.NewReader(tc.content)

			err := parser.LoadFromReader(reader)
			if err != nil {
				t.Fatalf("error loading from reader: %v", err)
			}

			if len(parser.patterns) != len(tc.expected) {
				t.Fatalf("pattern count is wrong: expected=%d, actual=%d",
					len(tc.expected), len(parser.patterns))
			}

			for i, expected := range tc.expected {
				actual := parser.patterns[i]
				if actual.Pattern != expected.Pattern {
					t.Errorf("pattern[%d] is wrong: expected='%s', actual='%s'",
						i, expected.Pattern, actual.Pattern)
				}

				if actual.Negate != expected.Negate {
					t.Errorf("negate flag[%d] is wrong: expected=%t, actual=%t",
						i, expected.Negate, actual.Negate)
				}
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	// Test: Check if file path should be ignored
	testCases := []struct {
		name     string
		patterns string
		filePath string
		expected bool
	}{
		{
			name:     "no patterns",
			patterns: "",
			filePath: "main.go",
			expected: false,
		},
		{
			name:     "wildcard match",
			patterns: "*.log",
			filePath: "app.log",
			expected: true,
		},
		{
			name:     "directory match",
			patterns: "vendor/",
			filePath: "vendor/package/file.go",
			expected: true,
		},
		{
			name:     "exact match",
			patterns: "main.go",
			filePath: "main.go",
			expected: true,
		},
		{
			name:     "basename match",
			patterns: "config.json",
			filePath: "app/config.json",
			expected: true,
		},
		{
			name: "negation pattern",
			patterns: `*.go
!important.go`,
			filePath: "important.go",
			expected: false,
		},
		{
			name: "complex negation pattern",
			patterns: `vendor/
!vendor/important/`,
			filePath: "vendor/important/file.go",
			expected: false,
		},
		{
			name:     "subdirectory not matched by root pattern",
			patterns: "testdata/",
			filePath: "internal/testdata/sample.go",
			expected: false,
		},
		{
			name:     "no pattern match",
			patterns: "*.log",
			filePath: "main.go",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := New()
			reader := strings.NewReader(tc.patterns)

			err := parser.LoadFromReader(reader)
			if err != nil {
				t.Fatalf("error loading patterns: %v", err)
			}

			result := parser.ShouldIgnore(tc.filePath)
			if result != tc.expected {
				t.Errorf("result is wrong: path='%s', expected=%t, actual=%t",
					tc.filePath, tc.expected, result)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Test: Load from .gomuignore file

	// 1. Create temporary directory
	tempDir := t.TempDir()
	ignoreFile := filepath.Join(tempDir, ".gomuignore")

	// 2. Create test .gomuignore file
	content := `# Go mutation testing ignore patterns
*.log
vendor/
testdata/
!important.go`

	err := os.WriteFile(ignoreFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create .gomuignore file: %v", err)
	}

	// 3. Load file with parser
	parser := New()

	err = parser.LoadFromFile(ignoreFile)
	if err != nil {
		t.Fatalf("error loading file: %v", err)
	}

	// 4. Verify expected patterns
	expectedPatterns := 4
	if len(parser.patterns) != expectedPatterns {
		t.Errorf("pattern count is wrong: expected=%d, actual=%d",
			expectedPatterns, len(parser.patterns))
	}

	// 5. Test specific patterns
	testCases := []struct {
		filePath string
		expected bool
	}{
		{"app.log", true},
		{"vendor/package/file.go", true},
		{"testdata/sample.go", true},
		{"important.go", false},
		{"main.go", false},
	}

	for _, tc := range testCases {
		result := parser.ShouldIgnore(tc.filePath)
		if result != tc.expected {
			t.Errorf("result for file '%s' is wrong: expected=%t, actual=%t",
				tc.filePath, tc.expected, result)
		}
	}
}

func TestLoadFromFileNotExists(t *testing.T) {
	// Test: Loading non-existent file (should not error)
	parser := New()
	err := parser.LoadFromFile("/nonexistent/path/.gomuignore")

	// Non-existent file should not cause error
	if err != nil {
		t.Errorf("non-existent file should not cause error: %v", err)
	}

	if len(parser.patterns) != 0 {
		t.Errorf("patterns should be empty: actual=%d", len(parser.patterns))
	}
}

func TestFindIgnoreFile(t *testing.T) {
	// Test: Find .gomuignore file

	// 1. Create nested directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub", "nested")

	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// 2. Create .gomuignore file in root directory
	ignoreFile := filepath.Join(tempDir, ".gomuignore")

	err = os.WriteFile(ignoreFile, []byte("*.log"), 0644)
	if err != nil {
		t.Fatalf("failed to create .gomuignore file: %v", err)
	}

	// 3. Search from nested directory
	foundFile, err := FindIgnoreFile(subDir)
	if err != nil {
		t.Fatalf("error searching file: %v", err)
	}

	// 4. Verify correct file was found
	if foundFile != ignoreFile {
		t.Errorf("found file is wrong: expected='%s', actual='%s'",
			ignoreFile, foundFile)
	}
}

func TestFindIgnoreFileNotFound(t *testing.T) {
	// Test: .gomuignore file not found
	tempDir := t.TempDir()

	foundFile, err := FindIgnoreFile(tempDir)
	if err != nil {
		t.Fatalf("error searching file: %v", err)
	}

	if foundFile != "" {
		t.Errorf("should return empty string when file not found: actual='%s'", foundFile)
	}
}

func TestMatchPattern(t *testing.T) {
	// Test: Detailed pattern matching tests
	parser := New()

	testCases := []struct {
		pattern  string
		filePath string
		expected bool
		desc     string
	}{
		{"*.go", "main.go", true, "wildcard (basename)"},
		{"*.go", "src/main.go", true, "wildcard (full path)"},
		{"vendor/", "vendor/", true, "directory pattern (exact match)"},
		{"vendor/", "vendor/pkg/file.go", true, "directory pattern (subdirectory)"},
		{"src/main.go", "project/src/main.go", true, "path suffix match"},
		{"config.json", "app/config/config.json", true, "basename match"},
		{"test/", "src/test/file.go", false, "subdirectory not matched"},
		{"*.txt", "main.go", false, "wildcard (no match)"},
		{"exact.go", "different.go", false, "exact match (no match)"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := parser.matchPattern(tc.pattern, tc.filePath)
			if result != tc.expected {
				t.Errorf("pattern match is wrong: pattern='%s', path='%s', expected=%t, actual=%t",
					tc.pattern, tc.filePath, tc.expected, result)
			}
		})
	}
}

func TestGetPatterns(t *testing.T) {
	// Test: Get patterns
	parser := New()
	reader := strings.NewReader(`*.go
vendor/
!important.go`)

	err := parser.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("error loading patterns: %v", err)
	}

	patterns := parser.GetPatterns()
	if len(patterns) != 3 {
		t.Errorf("pattern count is wrong: expected=3, actual=%d", len(patterns))
	}

	// Verify pattern contents
	expected := []struct {
		pattern string
		negate  bool
	}{
		{"*.go", false},
		{"vendor/", false},
		{"important.go", true},
	}

	for i, exp := range expected {
		if patterns[i].Pattern != exp.pattern {
			t.Errorf("pattern[%d] is wrong: expected='%s', actual='%s'",
				i, exp.pattern, patterns[i].Pattern)
		}

		if patterns[i].Negate != exp.negate {
			t.Errorf("negate flag[%d] is wrong: expected=%t, actual=%t",
				i, exp.negate, patterns[i].Negate)
		}
	}
}
