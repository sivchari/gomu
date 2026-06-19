package gomu

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

func TestWriteDryRunFile(t *testing.T) {
	var buf bytes.Buffer

	mutants := []mutation.Mutant{
		{Line: 12, Column: 7, Type: "arithmetic_binary", Description: "Replace + with -"},
		{Line: 20, Column: 3, Type: "conditional_binary", Description: "Replace < with <="},
	}

	writeDryRunFile(&buf, "foo.go", mutants)

	out := buf.String()
	for _, want := range []string{
		"foo.go (2 mutants)",
		"L12:7", "arithmetic_binary", "Replace + with -",
		"L20:3", "conditional_binary", "Replace < with <=",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output missing %q\n%s", want, out)
		}
	}
}

func TestDryRunNoFiles(t *testing.T) {
	engine, err := NewEngine(nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	var buf bytes.Buffer
	if err := engine.dryRun(&buf, nil); err != nil {
		t.Fatalf("dryRun: %v", err)
	}

	if !strings.Contains(buf.String(), "no files to analyze") {
		t.Errorf("unexpected output for empty file list: %q", buf.String())
	}
}

func TestDryRunGeneratesMutants(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)

	file := filepath.Join(tempDir, "math.go")
	writeFile(t, file, `package main

func Add(a, b int) int {
	return a + b
}
`)

	engine, err := NewEngine(nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	var buf bytes.Buffer
	if err := engine.dryRun(&buf, []string{file}); err != nil {
		t.Fatalf("dryRun: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"math.go", "Total:", "arithmetic"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output missing %q\n%s", want, out)
		}
	}
}

// TestRunDryRunSkipsExecution proves the dry-run path stops before execution and
// reporting: with JSON output configured, a real run would write
// mutation-report.json, but a dry run must not.
func TestRunDryRunSkipsExecution(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, filepath.Join(tempDir, "go.mod"), testModuleContent)
	writeFile(t, filepath.Join(tempDir, "math.go"), `package main

func Add(a, b int) int {
	return a + b
}
`)

	// Reports and history are written relative to the working directory; run
	// inside the isolated temp dir so any side effects are contained and
	// observable.
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	defer func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()

	opts := &RunOptions{DryRun: true, Incremental: false, Output: "json", Workers: 1, Timeout: 5}

	engine, err := NewEngine(opts)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	if err := engine.Run(context.Background(), ".", opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, name := range []string{"mutation-report.json", ".gomu_history.json"} {
		if _, err := os.Stat(filepath.Join(tempDir, name)); !os.IsNotExist(err) {
			t.Errorf("dry run should not create %s", name)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
