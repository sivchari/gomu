package mutation

import "testing"

// TestSupportedMutators verifies the catalog exposed for listing is complete:
// every registered mutator must report a unique, non-empty name and a non-empty
// description. This guards against a newly added mutator forgetting to
// implement Description().
func TestSupportedMutators(t *testing.T) {
	mutators := SupportedMutators()
	if len(mutators) == 0 {
		t.Fatal("SupportedMutators returned no mutators")
	}

	seen := make(map[string]bool, len(mutators))

	for _, m := range mutators {
		name := m.Name()
		if name == "" {
			t.Errorf("mutator %T has an empty Name()", m)
		}

		if m.Description() == "" {
			t.Errorf("mutator %q (%T) has an empty Description()", name, m)
		}

		if seen[name] {
			t.Errorf("duplicate mutator name %q", name)
		}

		seen[name] = true
	}
}
