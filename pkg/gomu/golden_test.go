package gomu

import (
	"embed"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

//go:embed testdata
var testdataFS embed.FS

var update = flag.Bool("update", false, "rewrite golden files under testdata/")

// assertGolden compares got against the golden file at relPath, or rewrites it
// under -update.
//
// The rewrite does not compare: the embedded testdataFS is only refreshed on
// the next build, so a rewritten file is stale until then.
func assertGolden(t *testing.T, relPath string, got []byte) {
	t.Helper()

	if *update {
		if err := os.WriteFile(relPath, got, 0o600); err != nil {
			t.Fatalf("failed to write golden file %s: %v", relPath, err)
		}

		t.Logf("wrote %s", relPath)

		return
	}

	want, err := testdataFS.ReadFile(filepath.ToSlash(relPath))
	if err != nil {
		t.Fatalf("missing golden file %s (run: go test ./pkg/gomu -update): %v", relPath, err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", relPath, diff)
	}
}
