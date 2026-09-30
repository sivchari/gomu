package execution

import (
	"os"
	"path/filepath"
	"testing"
)

// newTestModule writes a minimal Go module (go.mod plus the given files) into
// a fresh temp directory and returns its path. Fixture content is written
// byte-for-byte since callers reference exact line:column positions in it.
func newTestModule(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o600); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)

		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("failed to create dir for %s: %v", name, err)
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	return dir
}
