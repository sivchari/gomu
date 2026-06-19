package gomu

import (
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

func TestSupportedMutators(t *testing.T) {
	infos := SupportedMutators()

	if want := len(mutation.SupportedMutators()); len(infos) != want {
		t.Fatalf("expected %d mutators, got %d", want, len(infos))
	}

	for _, info := range infos {
		if info.Name == "" || info.Description == "" {
			t.Errorf("incomplete mutator info: %+v", info)
		}
	}
}
