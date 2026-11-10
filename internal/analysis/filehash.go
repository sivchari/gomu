// Package analysis provides file analysis and hashing functionality.
package analysis

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileHasher provides file hashing functionality.
type FileHasher struct{}

// NewFileHasher creates a new file hasher.
func NewFileHasher() *FileHasher {
	return &FileHasher{}
}

// HashFile calculates the SHA256 hash of a file.
func (h *FileHasher) HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// HashFiles calculates hashes for multiple files.
func (h *FileHasher) HashFiles(filePaths []string) (map[string]string, error) {
	hashes := make(map[string]string)

	for _, filePath := range filePaths {
		hash, err := h.HashFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to hash file %s: %w", filePath, err)
		}

		hashes[filePath] = hash
	}

	return hashes, nil
}

// HashRelatedTests calculates the combined hash of every *_test.go file in the package directory of filePath.
//
// It is the single source of truth for the TestHash stored in history and the one compared
// against it. File names are part of the hash so that adding, deleting or renaming a test file
// changes it. It returns "" when the directory has no hashable test file.
func (h *FileHasher) HashRelatedTests(filePath string) string {
	testFiles := packageTestFiles(filePath)
	combined := make([]byte, 0, len(testFiles)*sha256.Size*2)

	for _, testFile := range testFiles {
		hash, err := h.HashFile(testFile)
		if err != nil {
			continue
		}

		combined = append(combined, filepath.Base(testFile)...)
		combined = append(combined, 0)
		combined = append(combined, hash...)
	}

	if len(combined) == 0 {
		return ""
	}

	return h.HashContent(combined)
}

// HashContent calculates the SHA256 hash of content.
func (h *FileHasher) HashContent(content []byte) string {
	hash := sha256.New()
	hash.Write(content)

	return fmt.Sprintf("%x", hash.Sum(nil))
}
