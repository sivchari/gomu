package mutation

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const switchIssueExampleSrc = `package main

func Check(err error) error {
	switch err {
	case nil:
	default:
		return err
	}
	return nil
}
`

func findCaseClauses(t *testing.T, src string) (*ast.File, *token.FileSet, []*ast.CaseClause) {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	var clauses []*ast.CaseClause

	ast.Inspect(file, func(n ast.Node) bool {
		if cc, ok := n.(*ast.CaseClause); ok {
			clauses = append(clauses, cc)
		}

		return true
	})

	return file, fset, clauses
}

func TestSwitchMutator_Name(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}

	if mutator.Name() != switchMutatorName {
		t.Errorf("Name() = %q, want %q", mutator.Name(), switchMutatorName)
	}
}

func TestSwitchMutator_CanMutate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		expected []bool
	}{
		{
			name:     "nil/default example",
			src:      switchIssueExampleSrc,
			expected: []bool{false, true}, // case nil: (empty), default: return err
		},
		{
			name:     "default with body",
			src:      "package main\nfunc f() {\n\tswitch {\n\tdefault:\n\t\tprintln(1)\n\t}\n}",
			expected: []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutator := &SwitchMutator{}
			_, _, clauses := findCaseClauses(t, tt.src)

			if len(clauses) != len(tt.expected) {
				t.Fatalf("found %d case clauses, want %d", len(clauses), len(tt.expected))
			}

			for i, clause := range clauses {
				if got := mutator.CanMutate(clause); got != tt.expected[i] {
					t.Errorf("clause %d: CanMutate() = %v, want %v", i, got, tt.expected[i])
				}
			}
		})
	}
}

func TestSwitchMutator_CanMutate_NonCaseClauseNode(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutator.CanMutate(expr) {
		t.Error("CanMutate() = true, want false for non-CaseClause node")
	}
}

func TestSwitchMutator_Mutate_IssueExample(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}
	_, fset, clauses := findCaseClauses(t, switchIssueExampleSrc)

	var allMutants []Mutant

	for _, clause := range clauses {
		allMutants = append(allMutants, mutator.Mutate(clause, fset)...)
	}

	if len(allMutants) != 1 {
		t.Fatalf("Expected 1 mutant (only the default clause is non-empty), got %d", len(allMutants))
	}

	m := allMutants[0]

	if m.Type != switchCaseBodyRemovalType {
		t.Errorf("Type = %q, want %q", m.Type, switchCaseBodyRemovalType)
	}

	if m.Original != "default" {
		t.Errorf("Original = %q, want %q", m.Original, "default")
	}

	if m.Mutated == "" {
		t.Error("Expected non-empty Mutated")
	}

	if m.Description == "" {
		t.Error("Expected non-empty Description")
	}

	if m.Line <= 0 {
		t.Errorf("Expected positive line number, got %d", m.Line)
	}
}

func TestSwitchMutator_Mutate_TaglessSwitch(t *testing.T) {
	t.Parallel()

	src := `package main

func classify(x int) string {
	switch {
	case x > 0:
		return "positive"
	case x < 0:
		return "negative"
	default:
		return "zero"
	}
}
`

	mutator := &SwitchMutator{}
	_, fset, clauses := findCaseClauses(t, src)

	if len(clauses) != 3 {
		t.Fatalf("found %d case clauses, want 3", len(clauses))
	}

	wantOriginals := []string{"case x > 0", "case x < 0", "default"}

	type posKey struct {
		line, col int
	}

	seen := make(map[posKey]bool)

	for i, clause := range clauses {
		mutants := mutator.Mutate(clause, fset)
		if len(mutants) != 1 {
			t.Fatalf("clause %d: Expected 1 mutant, got %d", i, len(mutants))
		}

		m := mutants[0]

		if m.Original != wantOriginals[i] {
			t.Errorf("clause %d: Original = %q, want %q", i, m.Original, wantOriginals[i])
		}

		key := posKey{m.Line, m.Column}
		if seen[key] {
			t.Errorf("clause %d: duplicate line:column %v", i, key)
		}

		seen[key] = true
	}
}

func TestSwitchMutator_Mutate_MultiValueCase(t *testing.T) {
	t.Parallel()

	src := "package main\nfunc f(x int) {\n\tswitch x {\n\tcase 1, 2:\n\t\tprintln(x)\n\t}\n}"

	mutator := &SwitchMutator{}
	_, fset, clauses := findCaseClauses(t, src)

	if len(clauses) != 1 {
		t.Fatalf("found %d case clauses, want 1", len(clauses))
	}

	mutants := mutator.Mutate(clauses[0], fset)
	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant, got %d", len(mutants))
	}

	if want := "case 1, 2"; mutants[0].Original != want {
		t.Errorf("Original = %q, want %q", mutants[0].Original, want)
	}
}

func TestSwitchMutator_Mutate_EmptyClause(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}
	_, fset, clauses := findCaseClauses(t, switchIssueExampleSrc)

	// clauses[0] is "case nil:" with no body.
	if mutants := mutator.Mutate(clauses[0], fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for empty clause body", mutants)
	}
}

func TestSwitchMutator_Mutate_NonCaseClauseNode(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}
	fset := token.NewFileSet()

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	if mutants := mutator.Mutate(expr, fset); mutants != nil {
		t.Errorf("Mutate() = %v, want nil for non-CaseClause node", mutants)
	}
}

func TestSwitchMutator_Mutate_Fallthrough(t *testing.T) {
	t.Parallel()

	src := `package main

func classify(x int) string {
	switch x {
	case 1:
		fallthrough
	case 2:
		return "small"
	default:
		return "large"
	}
}
`

	file, fset, clauses := findCaseClauses(t, src)
	if len(clauses) != 3 {
		t.Fatalf("found %d case clauses, want 3", len(clauses))
	}

	mutator := &SwitchMutator{}

	// clauses[0] is "case 1:" ending with fallthrough.
	mutants := mutator.Mutate(clauses[0], fset)
	if len(mutants) != 1 {
		t.Fatalf("Expected 1 mutant for fallthrough clause, got %d", len(mutants))
	}

	if !mutator.Apply(clauses[0], mutants[0]) {
		t.Fatal("Apply() = false, want true")
	}

	if len(clauses[0].Body) != 0 {
		t.Errorf("len(Body) = %d, want 0 after Apply", len(clauses[0].Body))
	}

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		t.Fatalf("format.Node failed on mutated file: %v", err)
	}

	if _, err := parser.ParseFile(token.NewFileSet(), "mutated.go", buf.String(), 0); err != nil {
		t.Fatalf("mutated source is not valid Go: %v\n%s", err, buf.String())
	}
}

func TestSwitchMutator_Apply(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}
	_, _, clauses := findCaseClauses(t, switchIssueExampleSrc)

	defaultClause := clauses[1]

	mutant := Mutant{Type: switchCaseBodyRemovalType}

	if !mutator.Apply(defaultClause, mutant) {
		t.Error("Apply() = false, want true")
	}

	if len(defaultClause.Body) != 0 {
		t.Errorf("len(Body) = %d, want 0", len(defaultClause.Body))
	}
}

func TestSwitchMutator_Apply_WrongType(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}
	_, _, clauses := findCaseClauses(t, switchIssueExampleSrc)

	mutant := Mutant{Type: "unknown_type"}

	if mutator.Apply(clauses[1], mutant) {
		t.Error("Apply() = true, want false for unknown mutation type")
	}
}

func TestSwitchMutator_Apply_NonCaseClauseNode(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}

	expr, err := parser.ParseExpr("a && b")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	mutant := Mutant{Type: switchCaseBodyRemovalType}

	if mutator.Apply(expr, mutant) {
		t.Error("Apply() = true, want false for non-CaseClause node")
	}
}

func TestSwitchMutator_Apply_EmptyClauseBody(t *testing.T) {
	t.Parallel()

	mutator := &SwitchMutator{}
	_, _, clauses := findCaseClauses(t, switchIssueExampleSrc)

	mutant := Mutant{Type: switchCaseBodyRemovalType}

	if mutator.Apply(clauses[0], mutant) {
		t.Error("Apply() = true, want false for already-empty clause body")
	}
}

// TestGenerateMutants_SwitchCaseBodyRemoval is an end-to-end check that
// Engine.GenerateMutants surfaces switch_case_body_removal mutants for the
// issue's nil/default switch form.
func TestGenerateMutants_SwitchCaseBodyRemoval(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte(switchIssueExampleSrc), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	engine, err := New()
	if err != nil {
		t.Fatalf("Failed to create mutation engine: %v", err)
	}

	mutants, err := engine.GenerateMutants(testFile)
	if err != nil {
		t.Fatalf("Failed to generate mutants: %v", err)
	}

	var switchMutants []Mutant

	for _, m := range mutants {
		if m.Type == switchCaseBodyRemovalType {
			switchMutants = append(switchMutants, m)
		}
	}

	if len(switchMutants) != 1 {
		t.Fatalf("Expected 1 switch_case_body_removal mutant, got %d: %+v", len(switchMutants), switchMutants)
	}

	if switchMutants[0].Original != "default" {
		t.Errorf("Original = %q, want %q", switchMutants[0].Original, "default")
	}
}
