package history

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sivchari/gomu/internal/mutation"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "test_history.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected store to be non-nil")
	}

	if store.filepath != historyFile {
		t.Errorf("Expected filepath %s, got %s", historyFile, store.filepath)
	}

	if store.entries == nil {
		t.Error("Expected entries map to be initialized")
	}
}

func TestNew_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "existing_history.json")

	// Create existing history file
	existingData := `{
		"entries": {
			"test.go": {
				"fileHash": "hash123",
				"mutationScore": 85.5,
				"timestamp": "2023-01-01T00:00:00Z"
			}
		},
		"savedAt": "2023-01-01T00:00:00Z",
		"version": "v0.0.0"
	}`

	err := os.WriteFile(historyFile, []byte(existingData), 0600)
	if err != nil {
		t.Fatalf("Failed to write existing history file: %v", err)
	}

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store with existing file: %v", err)
	}

	if len(store.entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(store.entries))
	}

	entry, exists := store.entries["test.go"]
	if !exists {
		t.Error("Expected entry for test.go to exist")
	}

	if entry.FileHash != "hash123" {
		t.Errorf("Expected file hash 'hash123', got %s", entry.FileHash)
	}

	if entry.MutationScore != 85.5 {
		t.Errorf("Expected mutation score 85.5, got %f", entry.MutationScore)
	}
}

func TestNew_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "invalid_history.json")

	// Create invalid JSON file
	invalidData := `{
		"entries": {
			"test.go": invalid json
		}
	}`

	err := os.WriteFile(historyFile, []byte(invalidData), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid history file: %v", err)
	}

	_, err = New(historyFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestUpdateFile(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "update_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create test mutants and results
	mutants := []mutation.Mutant{
		{
			ID:          "test1",
			FilePath:    "test.go",
			Line:        10,
			Column:      5,
			Type:        "arithmetic",
			Original:    "+",
			Mutated:     "-",
			Description: "Replace + with -",
		},
		{
			ID:          "test2",
			FilePath:    "test.go",
			Line:        15,
			Column:      8,
			Type:        "conditional",
			Original:    "==",
			Mutated:     "!=",
			Description: "Replace == with !=",
		},
	}

	results := []mutation.Result{
		{
			Mutant: mutants[0],
			Status: mutation.StatusKilled,
		},
		{
			Mutant: mutants[1],
			Status: mutation.StatusSurvived,
		},
	}

	store.UpdateFile("test.go", mutants, results)

	entry, exists := store.GetEntry("test.go")
	if !exists {
		t.Fatal("Expected entry for test.go to exist")
	}

	if len(entry.Mutants) != 2 {
		t.Errorf("Expected 2 mutants, got %d", len(entry.Mutants))
	}

	if len(entry.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(entry.Results))
	}

	// Check mutation score calculation (1 killed out of 2 = 50%)
	expectedScore := 50.0
	if entry.MutationScore != expectedScore {
		t.Errorf("Expected mutation score %f, got %f", expectedScore, entry.MutationScore)
	}

	if entry.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestUpdateFile_AllKilled(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "all_killed_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mutants := []mutation.Mutant{
		{ID: "test1", FilePath: "test.go"},
		{ID: "test2", FilePath: "test.go"},
	}

	results := []mutation.Result{
		{Mutant: mutants[0], Status: mutation.StatusKilled},
		{Mutant: mutants[1], Status: mutation.StatusKilled},
	}

	store.UpdateFile("test.go", mutants, results)

	entry, _ := store.GetEntry("test.go")

	// Check mutation score (2 killed out of 2 = 100%)
	expectedScore := 100.0
	if entry.MutationScore != expectedScore {
		t.Errorf("Expected mutation score %f, got %f", expectedScore, entry.MutationScore)
	}
}

func TestUpdateFile_NoResults(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "no_results_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	store.UpdateFile("test.go", []mutation.Mutant{}, []mutation.Result{})

	entry, _ := store.GetEntry("test.go")

	// Check mutation score (no results = 0%)
	expectedScore := 0.0
	if entry.MutationScore != expectedScore {
		t.Errorf("Expected mutation score %f, got %f", expectedScore, entry.MutationScore)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "save_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add some data
	mutants := []mutation.Mutant{{ID: "test1", FilePath: "test.go"}}
	results := []mutation.Result{{Mutant: mutants[0], Status: mutation.StatusKilled}}
	store.UpdateFile("test.go", mutants, results)

	err = store.Save()
	if err != nil {
		t.Fatalf("Failed to save store: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Error("History file was not created")
	}

	// Load and verify content
	newStore, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to load saved store: %v", err)
	}

	if len(newStore.entries) != 1 {
		t.Errorf("Expected 1 entry in loaded store, got %d", len(newStore.entries))
	}

	entry, exists := newStore.GetEntry("test.go")
	if !exists {
		t.Error("Expected entry for test.go in loaded store")
	}

	if entry.MutationScore != 100.0 {
		t.Errorf("Expected mutation score 100.0, got %f", entry.MutationScore)
	}
}

func TestHasChanged(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "changed_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add entry with specific hash
	mutants := []mutation.Mutant{{ID: "test1", FilePath: "test.go"}}
	results := []mutation.Result{{Mutant: mutants[0], Status: mutation.StatusKilled}}
	store.UpdateFile("test.go", mutants, results)

	// Manually set the hash for testing
	entry := store.entries["test.go"]
	entry.FileHash = "hash123"
	store.entries["test.go"] = entry

	tests := []struct {
		name        string
		filePath    string
		currentHash string
		expected    bool
	}{
		{
			name:        "same hash",
			filePath:    "test.go",
			currentHash: "hash123",
			expected:    false,
		},
		{
			name:        "different hash",
			filePath:    "test.go",
			currentHash: "hash456",
			expected:    true,
		},
		{
			name:        "new file",
			filePath:    "new.go",
			currentHash: "hash789",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.HasChanged(tt.filePath, tt.currentHash)
			if result != tt.expected {
				t.Errorf("HasChanged() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "stats_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add multiple files with different scores
	// File 1: 100% score (2/2 killed)
	mutants1 := []mutation.Mutant{
		{ID: "test1", FilePath: "file1.go"},
		{ID: "test2", FilePath: "file1.go"},
	}
	results1 := []mutation.Result{
		{Mutant: mutants1[0], Status: mutation.StatusKilled},
		{Mutant: mutants1[1], Status: mutation.StatusKilled},
	}
	store.UpdateFile("file1.go", mutants1, results1)

	// File 2: 50% score (1/2 killed)
	mutants2 := []mutation.Mutant{
		{ID: "test3", FilePath: "file2.go"},
		{ID: "test4", FilePath: "file2.go"},
	}
	results2 := []mutation.Result{
		{Mutant: mutants2[0], Status: mutation.StatusKilled},
		{Mutant: mutants2[1], Status: mutation.StatusSurvived},
	}
	store.UpdateFile("file2.go", mutants2, results2)

	stats := store.GetStats()

	if stats.TotalFiles != 2 {
		t.Errorf("Expected TotalFiles 2, got %d", stats.TotalFiles)
	}

	if stats.TotalMutants != 4 {
		t.Errorf("Expected TotalMutants 4, got %d", stats.TotalMutants)
	}

	if stats.TotalKilled != 3 {
		t.Errorf("Expected TotalKilled 3, got %d", stats.TotalKilled)
	}

	// Average score should be (100 + 50) / 2 = 75
	expectedAvgScore := 75.0
	if stats.AverageScore != expectedAvgScore {
		t.Errorf("Expected AverageScore %f, got %f", expectedAvgScore, stats.AverageScore)
	}

	if stats.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}

func TestGetStats_EmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "empty_stats_test.json")

	store, err := New(historyFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	stats := store.GetStats()

	expectedStats := Stats{
		TotalFiles:   0,
		TotalMutants: 0,
		TotalKilled:  0,
		AverageScore: 0,
		LastUpdated:  stats.LastUpdated, // We'll check this separately
	}

	// Check individual fields (except LastUpdated)
	if stats.TotalFiles != expectedStats.TotalFiles {
		t.Errorf("Expected TotalFiles %d, got %d", expectedStats.TotalFiles, stats.TotalFiles)
	}

	if stats.TotalMutants != expectedStats.TotalMutants {
		t.Errorf("Expected TotalMutants %d, got %d", expectedStats.TotalMutants, stats.TotalMutants)
	}

	if stats.TotalKilled != expectedStats.TotalKilled {
		t.Errorf("Expected TotalKilled %d, got %d", expectedStats.TotalKilled, stats.TotalKilled)
	}

	if stats.AverageScore != expectedStats.AverageScore {
		t.Errorf("Expected AverageScore %f, got %f", expectedStats.AverageScore, stats.AverageScore)
	}

	if stats.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}
