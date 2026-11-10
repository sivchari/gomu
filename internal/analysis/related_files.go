package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// packageTestFiles returns the sorted paths of every *_test.go file in the directory of filePath.
//
// Tests exercise production code through the package API, so a test file does not need to be
// named after the file it covers. External test packages (package x_test) live in the same
// directory and are included.
func packageTestFiles(filePath string) []string {
	dir := filepath.Dir(filePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	testFiles := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		testFiles = append(testFiles, filepath.Join(dir, entry.Name()))
	}

	sort.Strings(testFiles)

	return testFiles
}
