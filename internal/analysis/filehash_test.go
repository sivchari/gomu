package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileHasher_HashFile(t *testing.T) {
	hasher := NewFileHasher()
	
	// Create temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test hashing
	hash1, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash file: %v", err)
	}
	
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
	
	// Hash same file again - should be identical
	hash2, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash file again: %v", err)
	}
	
	if hash1 != hash2 {
		t.Errorf("Hash should be identical for same file, got %s and %s", hash1, hash2)
	}
	
	// Modify file content
	modifiedContent := content + "\n// Modified"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0600); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}
	
	// Hash modified file - should be different
	hash3, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash modified file: %v", err)
	}
	
	if hash1 == hash3 {
		t.Error("Hash should be different for modified file")
	}
}

func TestFileHasher_HashFile_NonExistent(t *testing.T) {
	hasher := NewFileHasher()
	
	_, err := hasher.HashFile("/non/existent/file.go")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFileHasher_HashFiles(t *testing.T) {
	hasher := NewFileHasher()
	
	// Create temporary files
	tmpDir := t.TempDir()
	
	file1 := filepath.Join(tmpDir, "file1.go")
	file2 := filepath.Join(tmpDir, "file2.go")
	
	content1 := "package main\n\nfunc test1() {}"
	content2 := "package main\n\nfunc test2() {}"
	
	if err := os.WriteFile(file1, []byte(content1), 0600); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	
	if err := os.WriteFile(file2, []byte(content2), 0600); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	
	// Test hashing multiple files
	files := []string{file1, file2}
	hashes, err := hasher.HashFiles(files)
	if err != nil {
		t.Fatalf("Failed to hash files: %v", err)
	}
	
	if len(hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(hashes))
	}
	
	if hashes[file1] == "" {
		t.Error("Hash for file1 should not be empty")
	}
	
	if hashes[file2] == "" {
		t.Error("Hash for file2 should not be empty")
	}
	
	if hashes[file1] == hashes[file2] {
		t.Error("Hashes should be different for different files")
	}
}

func TestFileHasher_HashContent(t *testing.T) {
	hasher := NewFileHasher()
	
	content1 := []byte("Hello, World!")
	content2 := []byte("Hello, World!")
	content3 := []byte("Hello, Go!")
	
	hash1 := hasher.HashContent(content1)
	hash2 := hasher.HashContent(content2)
	hash3 := hasher.HashContent(content3)
	
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
	
	if hash1 != hash2 {
		t.Error("Hash should be same for identical content")
	}
	
	if hash1 == hash3 {
		t.Error("Hash should be different for different content")
	}
}