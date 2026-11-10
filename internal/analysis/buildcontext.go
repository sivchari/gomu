package analysis

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// BuildContext records which production files the active Go build
// configuration (build tags, GOOS/GOARCH, CGO_ENABLED, -race and so on)
// includes in each package under a directory tree.
//
// A nil *BuildContext imposes no restriction: every file is treated as active.
type BuildContext struct {
	// packageFiles maps a package directory to the absolute paths of its
	// compiled, non-test Go files.
	packageFiles map[string][]string
	// activeFiles is the set of all paths in packageFiles.
	activeFiles map[string]struct{}
}

// LoadBuildContext resolves, with the Go tool, the production files that the
// build configuration described by buildFlags and env includes for every
// package under dir.
//
// buildFlags are "go list"/"go build" flags such as -tags=unit. env is the
// process environment for the Go tool (nil means os.Environ()); it carries
// GOOS, GOARCH, CGO_ENABLED and similar settings.
func LoadBuildContext(dir string, buildFlags, env []string) (*BuildContext, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if env == nil {
		env = os.Environ()
	}

	cfg := &packages.Config{
		// NeedFiles is enough: GoFiles are the non-test files that pass the
		// build constraints, cgo files included. Nothing is compiled or
		// type-checked here.
		Mode:       packages.NeedName | packages.NeedFiles,
		Dir:        absDir,
		Env:        env,
		BuildFlags: buildFlags,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the active build configuration: %w", err)
	}

	bc := &BuildContext{
		packageFiles: make(map[string][]string, len(pkgs)),
		activeFiles:  make(map[string]struct{}),
	}

	for _, pkg := range pkgs {
		if len(pkg.GoFiles) == 0 && len(pkg.IgnoredFiles) == 0 {
			// Nothing known about this package's directory (for example a
			// package that failed to load): leave its files unrestricted.
			continue
		}

		pkgDir := filepath.Dir(firstNonEmpty(pkg.GoFiles, pkg.IgnoredFiles))
		bc.packageFiles[pkgDir] = append(bc.packageFiles[pkgDir], pkg.GoFiles...)

		for _, f := range pkg.GoFiles {
			bc.activeFiles[f] = struct{}{}
		}
	}

	return bc, nil
}

// firstNonEmpty returns the first element of the first non-empty list.
func firstNonEmpty(lists ...[]string) string {
	for _, l := range lists {
		if len(l) > 0 {
			return l[0]
		}
	}

	return ""
}

// IsActive reports whether path is part of the active build. A file is
// inactive only when it sits in a package directory the Go tool resolved and
// the active configuration does not include it. Files in directories the Go
// tool did not resolve are reported active, so discovery never loses files it
// has no information about.
func (b *BuildContext) IsActive(path string) bool {
	if b == nil {
		return true
	}

	if _, ok := b.packageFiles[filepath.Dir(path)]; !ok {
		return true
	}

	_, ok := b.activeFiles[path]

	return ok
}

// PackageFiles returns the active production files of the package in dir. The
// second result is false when dir is not a resolved package, in which case the
// caller has no active file list to restrict to.
func (b *BuildContext) PackageFiles(dir string) ([]string, bool) {
	if b == nil {
		return nil, false
	}

	files, ok := b.packageFiles[dir]

	return files, ok
}
