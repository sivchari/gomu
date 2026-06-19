package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sivchari/gomu/pkg/gomu"
)

func TestListMutators(t *testing.T) {
	var buf bytes.Buffer
	if err := listMutators(&buf); err != nil {
		t.Fatalf("listMutators: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Supported mutators") {
		t.Errorf("output missing header:\n%s", out)
	}

	mutators := gomu.SupportedMutators()
	if len(mutators) == 0 {
		t.Fatal("no supported mutators reported")
	}

	for _, m := range mutators {
		if !strings.Contains(out, m.Name) {
			t.Errorf("output missing mutator name %q:\n%s", m.Name, out)
		}

		if !strings.Contains(out, m.Description) {
			t.Errorf("output missing description %q:\n%s", m.Description, out)
		}
	}
}
