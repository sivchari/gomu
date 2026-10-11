package mutation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateMutantsSelectionSpans(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.go")

	source := `package example
//line virtual.go:100
func Updated(x int) int {
 if x > 0 {
  return x +
   2
 }
 return x
}
`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}

	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}

	mutants, err := engine.GenerateMutants(path)
	if err != nil {
		t.Fatal(err)
	}

	var arithmetic, branch, block bool

	for _, m := range mutants {
		if m.SelectionStartLine < 3 || m.SelectionEndLine > 9 || m.SelectionEndLine < m.SelectionStartLine {
			t.Errorf("invalid physical span: %+v", m)
		}

		if m.Line < 100 {
			t.Errorf("application position changed: %+v", m)
		}

		switch m.Type {
		case arithmeticBinaryType:
			arithmetic = true

			if m.SelectionStartLine != 5 || m.SelectionEndLine != 6 {
				t.Errorf("multiline expression span: %+v", m)
			}
		case branchConditionType:
			branch = true

			if m.SelectionStartLine != 4 || m.SelectionEndLine != 7 {
				t.Errorf("if span: %+v", m)
			}
		case emptyBlockType:
			block = true
		}

		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}

		for _, key := range []string{"SelectionStartLine", "SelectionEndLine", "selectionStartLine", "selectionEndLine"} {
			if _, ok := fields[key]; ok {
				t.Errorf("selection metadata serialized: %s", data)
			}
		}
	}

	if !arithmetic || !branch || !block {
		t.Errorf("missing mutants: arithmetic=%t branch=%t block=%t", arithmetic, branch, block)
	}
}
