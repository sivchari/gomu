// Package ignore handles .gomuignore file parsing and pattern matching.
package ignore

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Parser handles .gomuignore file parsing and pattern matching.
type Parser struct {
	patterns []Pattern
}

// Pattern represents an ignore pattern.
type Pattern struct {
	pattern string
	negate  bool // true if pattern starts with '!'
}

// New creates a new ignore parser.
func New() *Parser {
	return &Parser{
		patterns: make([]Pattern, 0),
	}
}

// LoadFromFile loads ignore patterns from a .gomuignore file.
func (p *Parser) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// .gomuignore file doesn't exist - this is not an error
			return nil
		}

		return fmt.Errorf("failed to open .gomuignore file: %w", err)
	}
	defer file.Close()

	return p.LoadFromReader(file)
}

// LoadFromReader loads ignore patterns from a reader.
func (p *Parser) LoadFromReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		pattern := Pattern{
			pattern: line,
			negate:  false,
		}

		// Handle negation patterns (starting with '!')
		if strings.HasPrefix(line, "!") {
			pattern.pattern = strings.TrimPrefix(line, "!")
			pattern.negate = true
		}

		p.patterns = append(p.patterns, pattern)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read patterns: %w", err)
	}

	return nil
}

// ShouldIgnore checks if a file path should be ignored based on loaded patterns.
func (p *Parser) ShouldIgnore(filePath string) bool {
	// Convert to forward slashes for consistent pattern matching
	normalizedPath := filepath.ToSlash(filePath)

	ignored := false

	// Process patterns in order
	for _, pattern := range p.patterns {
		matched := p.matchPattern(pattern.pattern, normalizedPath)

		if matched {
			if pattern.negate {
				// Negation pattern - don't ignore this file
				ignored = false
			} else {
				// Regular pattern - ignore this file
				ignored = true
			}
		}
	}

	return ignored
}

// matchPattern checks if a file path matches a pattern.
func (p *Parser) matchPattern(pattern, filePath string) bool {
	// Convert pattern to forward slashes
	pattern = filepath.ToSlash(pattern)

	// Handle directory patterns (ending with '/')
	if strings.HasSuffix(pattern, "/") {
		// Directory pattern - check if path starts with pattern
		dirPattern := strings.TrimSuffix(pattern, "/")

		return strings.HasPrefix(filePath, dirPattern+"/") || filePath == dirPattern
	}

	// Handle patterns with wildcards
	if strings.Contains(pattern, "*") {
		matched, err := filepath.Match(pattern, filepath.Base(filePath))
		if err == nil && matched {
			return true
		}

		// Also try matching the full path
		matched, err = filepath.Match(pattern, filePath)
		if err == nil && matched {
			return true
		}
	}

	// Exact match
	if pattern == filePath {
		return true
	}

	// Check if pattern matches any part of the path
	if strings.Contains(filePath, pattern) {
		return true
	}

	// Check if pattern matches the basename
	if pattern == filepath.Base(filePath) {
		return true
	}

	// Handle patterns like "dir/file.go"
	if strings.Contains(pattern, "/") && strings.HasSuffix(filePath, pattern) {
		return true
	}

	return false
}

// GetPatterns returns all loaded patterns for debugging purposes.
func (p *Parser) GetPatterns() []Pattern {
	return p.patterns
}

// FindIgnoreFile finds the .gomuignore file starting from the given directory
// and walking up to parent directories until found or reaching the root.
func FindIgnoreFile(startPath string) (string, error) {
	dir, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// If startPath is a file, get its directory
	if stat, err := os.Stat(dir); err == nil && !stat.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		ignoreFile := filepath.Join(dir, ".gomuignore")
		if _, err := os.Stat(ignoreFile); err == nil {
			return ignoreFile, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}

		dir = parent
	}

	return "", nil // Not found
}
