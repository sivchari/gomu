//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const mutatorTemplate = `package mutation

import (
	"fmt"
	"go/ast"
	"go/token"
)

const {{.LowerName}}MutatorName = "{{.LowerName}}"

// {{.StructName}}Mutator mutates {{.Description}}.
type {{.StructName}}Mutator struct {
}

// Name returns the name of the mutator.
func (m *{{.StructName}}Mutator) Name() string {
	return {{.LowerName}}MutatorName
}

// CanMutate returns true if the node can be mutated by this mutator.
func (m *{{.StructName}}Mutator) CanMutate(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		return m.is{{.StructName}}Op(n.Op)
	case *ast.UnaryExpr:
		// TODO: Add unary expression support if needed
		return false
	}

	return false
}

// Mutate generates mutants for the given node.
func (m *{{.StructName}}Mutator) Mutate(node ast.Node, fset *token.FileSet) []Mutant {
	var mutants []Mutant

	pos := fset.Position(node.Pos())

	switch n := node.(type) {
	case *ast.BinaryExpr:
		mutants = append(mutants, m.mutateBinaryExpr(n, pos)...)
	}

	return mutants
}

func (m *{{.StructName}}Mutator) mutateBinaryExpr(expr *ast.BinaryExpr, pos token.Position) []Mutant {
	mutations := m.get{{.StructName}}Mutations(expr.Op)

	mutants := make([]Mutant, 0, len(mutations))

	for _, newOp := range mutations {
		mutants = append(mutants, Mutant{
			Line:        pos.Line,
			Column:      pos.Column,
			Type:        "{{.LowerName}}_binary",
			Original:    expr.Op.String(),
			Mutated:     newOp.String(),
			Description: fmt.Sprintf("Replace %s with %s", expr.Op.String(), newOp.String()),
		})
	}

	return mutants
}

func (m *{{.StructName}}Mutator) is{{.StructName}}Op(op token.Token) bool {
	// TODO: Implement operator check
	switch op {
	// TODO: Add relevant token types
	// case token.AND:
	//     return true
	default:
		return false
	}
}

func (m *{{.StructName}}Mutator) get{{.StructName}}Mutations(op token.Token) []token.Token {
	// TODO: Implement mutations mapping
	switch op {
	// TODO: Add mutation mappings
	// case token.AND:
	//     return []token.Token{token.OR, token.XOR}
	default:
		return nil
	}
}
`

const testTemplate = `package mutation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func Test{{.StructName}}Mutator_Name(t *testing.T) {
	mutator := &{{.StructName}}Mutator{}
	if mutator.Name() != {{.LowerName}}MutatorName {
		t.Errorf("Expected name '%s', got %s", {{.LowerName}}MutatorName, mutator.Name())
	}
}

func Test{{.StructName}}Mutator_CanMutate(t *testing.T) {
	mutator := &{{.StructName}}Mutator{}

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "supported operation",
			code:     "a & b", // TODO: Update with actual supported operation
			expected: false,   // TODO: Update when implementation is complete
		},
		{
			name:     "unsupported operation",
			code:     "a + b",
			expected: false,
		},
		{
			name:     "function call",
			code:     "fmt.Println()",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			if canMutate := mutator.CanMutate(expr); canMutate != tt.expected {
				t.Errorf("CanMutate() = %v, expected %v", canMutate, tt.expected)
			}
		})
	}
}

func Test{{.StructName}}Mutator_Mutate_BinaryExpr(t *testing.T) {
	mutator := &{{.StructName}}Mutator{}
	fset := token.NewFileSet()

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "example operation",
			code:     "a & b", // TODO: Update with actual supported operation
			expected: []string{}, // TODO: Update with expected mutations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "package main\nfunc test() { _ = " + tt.code + " }"

			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Find the binary expression
			var expr ast.Expr
			ast.Inspect(file, func(node ast.Node) bool {
				if be, ok := node.(*ast.BinaryExpr); ok {
					expr = be
					return false
				}
				return true
			})

			if expr == nil {
				t.Fatalf("Binary expression not found in: %s", tt.code)
			}

			mutants := mutator.Mutate(expr, fset)

			if len(mutants) != len(tt.expected) {
				t.Errorf("Expected %d mutants, got %d", len(tt.expected), len(mutants))
			}

			// Check that all expected mutations are present
			mutatedOps := make(map[string]bool)
			for _, mutant := range mutants {
				mutatedOps[mutant.Mutated] = true
			}

			for _, expectedOp := range tt.expected {
				if !mutatedOps[expectedOp] {
					t.Errorf("Expected mutation to %s not found", expectedOp)
				}
			}

			// Check mutant properties
			for _, mutant := range mutants {
				if mutant.Type != "{{.LowerName}}_binary" {
					t.Errorf("Expected type '{{.LowerName}}_binary', got %s", mutant.Type)
				}

				if mutant.Line <= 0 {
					t.Errorf("Expected positive line number, got %d", mutant.Line)
				}

				if mutant.Description == "" {
					t.Error("Expected non-empty description")
				}
			}
		})
	}
}
`

type mutatorData struct {
	LowerName   string
	StructName  string
	Description string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run scaffold.go <mutator_name>\n")
		fmt.Fprintf(os.Stderr, "Example: go run scaffold.go bitwise\n")
		os.Exit(1)
	}

	name := strings.ToLower(os.Args[1])
	structName := strings.Title(name)

	data := mutatorData{
		LowerName:   name,
		StructName:  structName,
		Description: name + " operators",
	}

	// Generate mutator file
	mutatorFile := name + ".go"
	if err := generateFile(mutatorFile, mutatorTemplate, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating mutator file: %v\n", err)
		os.Exit(1)
	}

	// Generate test file
	testFile := name + "_test.go"
	if err := generateFile(testFile, testTemplate, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating test file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s mutator:\n", name)
	fmt.Printf("  - %s\n", mutatorFile)
	fmt.Printf("  - %s\n", testFile)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Update the TODO items in %s\n", mutatorFile)
	fmt.Printf("  2. Update the test cases in %s\n", testFile)
	fmt.Printf("  3. Run: go generate ./internal/mutation\n")
	fmt.Printf("  4. Run: go test ./internal/mutation\n")
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