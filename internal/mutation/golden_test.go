package mutation

import (
	"embed"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

//go:embed testdata
var testdataFS embed.FS

var update = flag.Bool("update", false, "rewrite golden files under testdata/")

const mutantsGoldenName = "mutants.golden"

// TestMutators drives one golden comparison per testdata/<case> directory:
// GenerateMutants must reproduce mutants.golden, and applying each of those
// mutants with ApplyMutantToSource must reproduce its NN.golden.
func TestMutators(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(testdataFS, "testdata")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runGoldenCase(t, name)
		})
	}
}

// runGoldenCase generates and compares the goldens for a single testdata case.
func runGoldenCase(t *testing.T, name string) {
	t.Helper()

	caseDir := filepath.Join("testdata", name)
	inputPath := filepath.Join(caseDir, "input.go")

	src, err := testdataFS.ReadFile(filepath.ToSlash(inputPath))
	if err != nil {
		t.Fatalf("failed to read %s: %v", inputPath, err)
	}

	formatted, err := format.Source(src)
	if err != nil {
		t.Fatalf("%s is not valid Go: %v", inputPath, err)
	}

	if string(formatted) != string(src) {
		t.Fatalf("%s is not gofmt-clean; run gofmt -w on it", inputPath)
	}

	engine, err := New()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(inputPath)
	if err != nil {
		t.Fatalf("failed to generate mutants for %s: %v", inputPath, err)
	}

	slices.SortStableFunc(mutants, compareMutants)

	var b strings.Builder

	for i, m := range mutants {
		fmt.Fprintf(&b, "%02d\t%d:%d\t%s\t%q -> %q\t%s\n", i+1, m.Line, m.Column, m.Type, m.Original, m.Mutated, m.Description)
	}

	assertGolden(t, filepath.Join(caseDir, mutantsGoldenName), []byte(b.String()))

	for i, m := range mutants {
		got, err := ApplyMutantToSource(src, "input.go", m)
		if err != nil {
			t.Errorf("mutant %02d: ApplyMutantToSource failed: %v", i+1, err)

			continue
		}

		assertGolden(t, filepath.Join(caseDir, fmt.Sprintf("%02d.golden", i+1)), got)
	}

	checkStaleGoldens(t, caseDir, len(mutants))
}

// compareMutants orders mutants deterministically, independent of registry order.
func compareMutants(a, b Mutant) int {
	if c := a.Line - b.Line; c != 0 {
		return c
	}

	if c := a.Column - b.Column; c != 0 {
		return c
	}

	if c := strings.Compare(a.Type, b.Type); c != 0 {
		return c
	}

	if c := strings.Compare(a.Original, b.Original); c != 0 {
		return c
	}

	if c := strings.Compare(a.Mutated, b.Mutated); c != 0 {
		return c
	}

	return strings.Compare(a.Description, b.Description)
}

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
		t.Fatalf("missing golden file %s (run: go test ./internal/mutation -run TestMutators -update): %v", relPath, err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", relPath, diff)
	}
}

// checkStaleGoldens fails (or, under -update, removes) any NN.golden file
// left over from a case that used to produce more than n mutants.
func checkStaleGoldens(t *testing.T, dir string, n int) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dir, err)
	}

	for _, entry := range entries {
		name := entry.Name()

		idx, ok := strings.CutSuffix(name, ".golden")
		if !ok || name == mutantsGoldenName {
			continue
		}

		num, convErr := strconv.Atoi(idx)
		if convErr == nil && num <= n {
			continue
		}

		path := filepath.Join(dir, name)

		if *update {
			if err := os.Remove(path); err != nil {
				t.Fatalf("failed to remove stale golden %s: %v", path, err)
			}

			t.Logf("removed stale golden %s", path)

			continue
		}

		t.Errorf("stale golden file %s: case now produces only %d mutants", path, n)
	}
}
