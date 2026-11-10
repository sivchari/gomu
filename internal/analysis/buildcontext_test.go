package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sivchari/gomu/internal/ignore"
)

// copyFixture copies a module under testdata into a temp dir and returns the
// copy, so tests can add files and run outside the repository's own module.
func copyFixture(t *testing.T, name string) string {
	t.Helper()

	dst := t.TempDir()
	if err := os.CopyFS(dst, os.DirFS(filepath.Join("testdata", name))); err != nil {
		t.Fatalf("copy fixture %s: %v", name, err)
	}

	return dst
}

// discoverTargets returns the base names of the files the shared discovery
// path (IncrementalAnalyzer, used by both normal and dry runs) selects.
func discoverTargets(t *testing.T, dir string, buildFlags, env []string, opts ...func(*IncrementalAnalyzer)) []string {
	t.Helper()

	bc, err := LoadBuildContext(dir, buildFlags, env)
	if err != nil {
		t.Fatalf("LoadBuildContext: %v", err)
	}

	inc, err := NewIncrementalAnalyzer(dir, noHistory{}, false, "main")
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer: %v", err)
	}

	inc.SetBuildContext(bc)

	for _, opt := range opts {
		opt(inc)
	}

	files, err := inc.GetFilesNeedingUpdate()
	if err != nil {
		t.Fatalf("GetFilesNeedingUpdate: %v", err)
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}

	sort.Strings(names)

	return names
}

type noHistory struct{}

func (noHistory) GetEntry(string) (HistoryEntry, bool) { return HistoryEntry{}, false }
func (noHistory) HasChanged(string, string) bool       { return true }

func TestDiscovery_BuildTags(t *testing.T) {
	tests := []struct {
		name       string
		buildFlags []string
		want       []string
	}{
		{"tag selects the tagged file", []string{"-tags=unit"}, []string{"active.go"}},
		{"separate tag value", []string{"-tags", "unit"}, []string{"active.go"}},
		{"no tag selects the untagged file", nil, []string{"inactive.go"}},
		// A tag that only matters to test files must not hide active production code.
		{"unrelated tag keeps default production file", []string{"-tags=integration"}, []string{"inactive.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := copyFixture(t, "buildtags")

			got := discoverTargets(t, dir, tt.buildFlags, nil)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("targets mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiscovery_IncrementalSelection(t *testing.T) {
	dir := copyFixture(t, "buildtags")

	bc, err := LoadBuildContext(dir, []string{"-tags=unit"}, nil)
	if err != nil {
		t.Fatalf("LoadBuildContext: %v", err)
	}

	runGit := newTestGitRunner(t, dir)
	runGit("init", "-b", "main")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	runGit("config", "commit.gpgsign", "false")
	runGit("add", ".")
	runGit("commit", "-m", "base")

	// Both production files change, but only active.go is in the build.
	commitFiles(t, dir, runGit, map[string]string{
		"active.go":   "//go:build unit\n\npackage example\n\nfunc Value() int { return 7 }\n",
		"inactive.go": "//go:build !unit\n\npackage example\n\nfunc Value() int { return 8 }\n",
	})

	git := NewGitIntegration(dir)
	git.SetBuildContext(bc)

	files, err := git.GetChangedFilesSince("HEAD~1")
	if err != nil {
		t.Fatalf("GetChangedFilesSince: %v", err)
	}

	if len(files) != 1 || filepath.Base(files[0]) != "active.go" {
		t.Errorf("changed files = %v, want only active.go", files)
	}
}

func TestDiscovery_PlatformAndRace(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/plat\n\ngo 1.24\n")
	writeTestFile(t, filepath.Join(dir, "common.go"), "package plat\n")
	writeTestFile(t, filepath.Join(dir, "only_linux.go"), "package plat\n")
	writeTestFile(t, filepath.Join(dir, "only_windows.go"), "package plat\n")
	writeTestFile(t, filepath.Join(dir, "race.go"), "//go:build race\n\npackage plat\n")

	tests := []struct {
		name       string
		buildFlags []string
		env        []string
		want       []string
	}{
		{"linux", nil, []string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0"}, []string{"common.go", "only_linux.go"}},
		{"windows", nil, []string{"GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0"}, []string{"common.go", "only_windows.go"}},
		{"race tag", []string{"-race"}, []string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=1"}, []string{"common.go", "only_linux.go", "race.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := append(os.Environ(), tt.env...)

			got := discoverTargets(t, dir, tt.buildFlags, env)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("targets mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiscovery_IgnoreAndGeneratedStillApply(t *testing.T) {
	dir := copyFixture(t, "buildtags")
	writeTestFile(t, filepath.Join(dir, "extra.go"), "//go:build unit\n\npackage example\n\nfunc Extra() int { return 1 }\n")
	writeTestFile(t, filepath.Join(dir, "skipped.go"), "//go:build unit\n\npackage example\n\nfunc Skipped() int { return 1 }\n")
	writeTestFile(t, filepath.Join(dir, "gen.go"), "// Code generated by test. DO NOT EDIT.\n\n//go:build unit\n\npackage example\n\nfunc Gen() int { return 1 }\n")
	writeTestFile(t, filepath.Join(dir, ".gomuignore"), "skipped.go\n")

	parser := ignore.New()
	if err := parser.LoadFromFile(filepath.Join(dir, ".gomuignore")); err != nil {
		t.Fatalf("load .gomuignore: %v", err)
	}

	flags := []string{"-tags=unit"}

	got := discoverTargets(t, dir, flags, nil, func(inc *IncrementalAnalyzer) { inc.SetIgnoreParser(parser) })
	if diff := cmp.Diff([]string{"active.go", "extra.go"}, got); diff != "" {
		t.Errorf("targets mismatch (-want +got):\n%s", diff)
	}

	got = discoverTargets(t, dir, flags, nil, func(inc *IncrementalAnalyzer) {
		inc.SetIgnoreParser(parser)
		inc.SetIncludeGenerated(true)
	})
	if diff := cmp.Diff([]string{"active.go", "extra.go", "gen.go"}, got); diff != "" {
		t.Errorf("targets with generated mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFile_TypeAnalysisUsesActivePackage(t *testing.T) {
	tests := []struct {
		name       string
		buildFlags []string
		wantType   string
	}{
		// Value is declared twice under opposite tags; combining both would
		// leave Got typed by whichever declaration sorts first.
		{"tag selects int Value", []string{"-tags=unit"}, "int"},
		{"no tag selects string Value", nil, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := copyFixture(t, "exclusive")

			bc, err := LoadBuildContext(dir, tt.buildFlags, nil)
			if err != nil {
				t.Fatalf("LoadBuildContext: %v", err)
			}

			a, err := New()
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			a.SetBuildContext(bc)

			info, err := a.ParseFile(filepath.Join(dir, "common.go"))
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}

			if info.TypeInfo == nil {
				t.Fatal("TypeInfo is nil")
			}

			if got := definedType(info, "Got"); got != tt.wantType {
				t.Errorf("type of Got = %q, want %q", got, tt.wantType)
			}
		})
	}
}

// TestParseFile_TypeAnalysisKeepsMutationExcludedFiles proves files excluded
// only from mutation (.gomuignore, generated) stay in the type-checked package.
func TestParseFile_TypeAnalysisKeepsMutationExcludedFiles(t *testing.T) {
	dir := copyFixture(t, "exclusive")
	writeTestFile(t, filepath.Join(dir, "gen.go"), "// Code generated by test. DO NOT EDIT.\n\npackage exclusive\n\nfunc Gen() float64 { return 1 }\n")
	writeTestFile(t, filepath.Join(dir, "use.go"), "package exclusive\n\nvar FromGen = Gen()\n")

	bc, err := LoadBuildContext(dir, []string{"-tags=unit"}, nil)
	if err != nil {
		t.Fatalf("LoadBuildContext: %v", err)
	}

	a, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a.SetBuildContext(bc)

	info, err := a.ParseFile(filepath.Join(dir, "use.go"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if got := definedType(info, "FromGen"); got != "float64" {
		t.Errorf("type of FromGen = %q, want float64", got)
	}
}

func TestBuildContext_NilIsUnrestricted(t *testing.T) {
	var bc *BuildContext

	if !bc.IsActive("/any/file.go") {
		t.Error("nil BuildContext must treat every file as active")
	}

	if _, ok := bc.PackageFiles("/any"); ok {
		t.Error("nil BuildContext must not claim a package file list")
	}
}

func TestBuildContext_UnresolvedDirectoryIsActive(t *testing.T) {
	dir := copyFixture(t, "buildtags")

	bc, err := LoadBuildContext(dir, []string{"-tags=unit"}, nil)
	if err != nil {
		t.Fatalf("LoadBuildContext: %v", err)
	}

	if !bc.IsActive(filepath.Join(t.TempDir(), "elsewhere.go")) {
		t.Error("a file outside every resolved package must stay active")
	}

	if bc.IsActive(filepath.Join(dir, "inactive.go")) {
		t.Error("inactive.go must be inactive under -tags=unit")
	}
}

func definedType(info *FileInfo, name string) string {
	for id, obj := range info.TypeInfo.Defs {
		if id.Name == name && obj != nil {
			return obj.Type().String()
		}
	}

	return ""
}
