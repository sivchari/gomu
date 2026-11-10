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

func TestFileHasher_HashRelatedTests(t *testing.T) {
	hasher := NewFileHasher()

	t.Run("one test file gives a deterministic non-empty hash", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "foo.go")
		testFile := filepath.Join(dir, "foo_test.go")

		writeTestFile(t, src, "package foo\n")
		writeTestFile(t, testFile, "package foo\n")

		first := hasher.HashRelatedTests(src)
		second := hasher.HashRelatedTests(src)

		if first == "" {
			t.Fatal("expected non-empty hash")
		}

		if first != second {
			t.Errorf("hash is not deterministic: %q != %q", first, second)
		}

		writeTestFile(t, testFile, "package foo\n// changed\n")

		if hasher.HashRelatedTests(src) == first {
			t.Error("hash should change when the test file changes")
		}
	})

	t.Run("hash changes when either of two test files changes", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "foo.go")
		suffixTest := filepath.Join(dir, "foo_test.go")
		otherTest := filepath.Join(dir, "contract_test.go")

		writeTestFile(t, src, "package foo\n")
		writeTestFile(t, suffixTest, "package foo\n")
		writeTestFile(t, otherTest, "package foo_test\n")

		base := hasher.HashRelatedTests(src)
		if base == "" {
			t.Fatal("expected non-empty hash")
		}

		writeTestFile(t, suffixTest, "package foo\n// a\n")

		afterFirst := hasher.HashRelatedTests(src)
		if afterFirst == base {
			t.Error("hash should change when foo_test.go changes")
		}

		writeTestFile(t, otherTest, "package foo_test\n// b\n")

		if hasher.HashRelatedTests(src) == afterFirst {
			t.Error("hash should change when contract_test.go changes")
		}
	})

	t.Run("a test file with an unrelated name invalidates every file in the package", func(t *testing.T) {
		dir := t.TempDir()
		value := filepath.Join(dir, "value.go")
		other := filepath.Join(dir, "other.go")
		contract := filepath.Join(dir, "contract_test.go")

		writeTestFile(t, value, "package example\n")
		writeTestFile(t, other, "package example\n")
		writeTestFile(t, contract, "package example_test\n")

		valueBefore := hasher.HashRelatedTests(value)
		otherBefore := hasher.HashRelatedTests(other)

		if valueBefore == "" {
			t.Fatal("expected contract_test.go to count as a test of value.go")
		}

		if valueBefore != otherBefore {
			t.Error("every production file in the package should share the package test hash")
		}

		writeTestFile(t, contract, "package example_test\n// changed\n")

		if hasher.HashRelatedTests(value) == valueBefore {
			t.Error("value.go hash should change when contract_test.go changes")
		}

		if hasher.HashRelatedTests(other) == otherBefore {
			t.Error("other.go hash should change when contract_test.go changes")
		}
	})

	t.Run("adding, deleting and renaming a test file changes the hash", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "foo.go")
		first := filepath.Join(dir, "a_test.go")
		second := filepath.Join(dir, "b_test.go")

		writeTestFile(t, src, "package foo\n")
		writeTestFile(t, first, "package foo\n")

		withOne := hasher.HashRelatedTests(src)

		writeTestFile(t, second, "package foo\n")

		withTwo := hasher.HashRelatedTests(src)
		if withTwo == withOne {
			t.Error("hash should change when a test file is added")
		}

		if err := os.Remove(second); err != nil {
			t.Fatalf("remove: %v", err)
		}

		if hasher.HashRelatedTests(src) != withOne {
			t.Error("hash should return to its previous value once the added file is deleted")
		}

		if err := os.Remove(first); err != nil {
			t.Fatalf("remove: %v", err)
		}

		if got := hasher.HashRelatedTests(src); got != "" {
			t.Errorf("hash should be empty once every test file is deleted, got %q", got)
		}

		writeTestFile(t, first, "package foo\n")

		renamed := filepath.Join(dir, "c_test.go")
		if err := os.Rename(first, renamed); err != nil {
			t.Fatalf("rename: %v", err)
		}

		if hasher.HashRelatedTests(src) == withOne {
			t.Error("hash should change when a test file is renamed, even with identical content")
		}
	})

	t.Run("files that are not Go test files are ignored", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "foo.go")

		writeTestFile(t, src, "package foo\n")
		writeTestFile(t, filepath.Join(dir, "test_foo.go"), "package foo\n")
		writeTestFile(t, filepath.Join(dir, "helper.go"), "package foo\n")
		writeTestFile(t, filepath.Join(dir, "notes_test.txt"), "x\n")

		if err := os.Mkdir(filepath.Join(dir, "sub_test.go"), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if got := hasher.HashRelatedTests(src); got != "" {
			t.Errorf("expected empty hash, got %q", got)
		}
	})

	t.Run("test files in other directories are ignored", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		src := filepath.Join(dir, "foo.go")

		if err := os.Mkdir(sub, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		writeTestFile(t, src, "package foo\n")
		writeTestFile(t, filepath.Join(sub, "foo_test.go"), "package sub\n")

		if got := hasher.HashRelatedTests(src); got != "" {
			t.Errorf("expected empty hash, got %q", got)
		}
	})

	t.Run("no test files gives empty hash", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "foo.go")

		writeTestFile(t, src, "package foo\n")

		if got := hasher.HashRelatedTests(src); got != "" {
			t.Errorf("expected empty hash, got %q", got)
		}
	})
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
