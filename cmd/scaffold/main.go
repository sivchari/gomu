package main

import (
	_ "embed"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

//go:embed templates/mutator.go.tmpl
var mutatorTemplate string

//go:embed templates/mutator_test.go.tmpl
var testTemplate string

type mutatorData struct {
	LowerName   string
	StructName  string
	Description string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <mutator_name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s bitwise\n", os.Args[0])
		os.Exit(1)
	}

	name := strings.ToLower(os.Args[1])
	structName := strings.Title(name)

	data := mutatorData{
		LowerName:   name,
		StructName:  structName,
		Description: name + " operators",
	}

	// Find the mutation package directory
	mutationDir, err := findMutationDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding mutation directory: %v\n", err)
		os.Exit(1)
	}

	// Generate mutator file
	mutatorFile := filepath.Join(mutationDir, name+".go")
	if err := generateFile(mutatorFile, mutatorTemplate, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating mutator file: %v\n", err)
		os.Exit(1)
	}

	// Generate test file
	testFile := filepath.Join(mutationDir, name+"_test.go")
	if err := generateFile(testFile, testTemplate, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating test file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s mutator:\n", name)
	fmt.Printf("  - %s\n", mutatorFile)
	fmt.Printf("  - %s\n", testFile)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Update the TODO items in %s\n", filepath.Base(mutatorFile))
	fmt.Printf("  2. Update the test cases in %s\n", filepath.Base(testFile))
	fmt.Printf("  3. Run: make generate-registry\n")
	fmt.Printf("  4. Run: make test\n")
}

func findMutationDir() (string, error) {
	// Try to find the mutation directory relative to current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// First check if we're already in the mutation directory
	if filepath.Base(wd) == "mutation" {
		return wd, nil
	}

	// Look for internal/mutation relative to current directory
	candidates := []string{
		filepath.Join(wd, "internal", "mutation"),
		filepath.Join(wd, "..", "internal", "mutation"),
		filepath.Join(wd, "..", "..", "internal", "mutation"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "engine.go")); err == nil {
			return candidate, nil
		}
	}

	// Try to find using Go module information
	pkg, err := build.Import("github.com/sivchari/gomu/internal/mutation", "", build.FindOnly)
	if err == nil {
		return pkg.Dir, nil
	}

	// Try using runtime caller information
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// cmd/scaffold/main.go -> ../../internal/mutation
		cmdDir := filepath.Dir(filepath.Dir(filename))
		mutationDir := filepath.Join(filepath.Dir(cmdDir), "internal", "mutation")
		if _, err := os.Stat(filepath.Join(mutationDir, "engine.go")); err == nil {
			return mutationDir, nil
		}
	}

	return "", fmt.Errorf("could not find mutation directory")
}

func generateFile(filename, tmplText string, data mutatorData) error {
	tmpl, err := template.New("mutator").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}