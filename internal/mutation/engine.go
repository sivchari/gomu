package mutation

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/sivchari/gomu/internal/analysis"
	"github.com/sivchari/gomu/internal/config"
)

// Engine handles mutation generation.
type Engine struct {
	config   *config.Config
	analyzer *analysis.Analyzer
	mutators []Mutator
}

// Mutant represents a single mutation.
type Mutant struct {
	ID          string `json:"id"`
	FilePath    string `json:"filePath"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Type        string `json:"type"`
	Original    string `json:"original"`
	Mutated     string `json:"mutated"`
	Description string `json:"description"`
}

// Result represents the result of testing a mutant.
type Result struct {
	Mutant Mutant `json:"mutant"`
	Status Status `json:"status"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Status represents the outcome of testing a mutant.
type Status string

const (
	// StatusKilled indicates that a mutant was detected by tests.
	StatusKilled Status = "KILLED" // Mutant was detected by tests
	// StatusSurvived indicates that a mutant was not detected by tests.
	StatusSurvived Status = "SURVIVED" // Mutant was not detected by tests
	// StatusTimedOut indicates that tests timed out.
	StatusTimedOut Status = "TIMED_OUT" // Tests timed out
	// StatusError indicates a build or runtime error.
	StatusError Status = "ERROR" // Build or runtime error
	// StatusNotCovered indicates that a mutant was not covered by tests.
	StatusNotCovered Status = "NOT_COVERED" // Mutant not covered by tests
)

// Mutator interface for different types of mutations.
type Mutator interface {
	Name() string
	CanMutate(node ast.Node) bool
	Mutate(node ast.Node, fset *token.FileSet) []Mutant
}

// New creates a new mutation engine.
func New(cfg *config.Config) (*Engine, error) {
	analyzer, err := analysis.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	engine := &Engine{
		config:   cfg,
		analyzer: analyzer,
		mutators: make([]Mutator, 0),
	}

	// Register mutators based on config
	for _, mutatorName := range cfg.Mutators {
		mutator := engine.createMutator(mutatorName)
		if mutator != nil {
			engine.mutators = append(engine.mutators, mutator)
		}
	}

	return engine, nil
}

func (e *Engine) createMutator(name string) Mutator {
	switch name {
	case "arithmetic":
		return &ArithmeticMutator{}
	case "conditional":
		return &ConditionalMutator{}
	case "logical":
		return &LogicalMutator{}
	default:
		return nil
	}
}

// GenerateMutants generates all possible mutants for a given file.
func (e *Engine) GenerateMutants(filePath string) ([]Mutant, error) {
	fileInfo, err := e.analyzer.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var allMutants []Mutant

	// Walk the AST and apply mutators
	ast.Inspect(fileInfo.FileAST, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		for _, mutator := range e.mutators {
			if mutator.CanMutate(node) {
				mutants := mutator.Mutate(node, e.analyzer.GetFileSet())
				for i := range mutants {
					mutants[i].FilePath = filePath
					mutants[i].ID = fmt.Sprintf("%s_%d", filePath, len(allMutants)+i)
				}

				allMutants = append(allMutants, mutants...)

				// Respect mutation limit
				if e.config.MutationLimit > 0 && len(allMutants) >= e.config.MutationLimit {
					return false
				}
			}
		}

		return true
	})

	return allMutants, nil
}

// GetFileSet returns the file set used by the engine.
func (e *Engine) GetFileSet() *token.FileSet {
	return e.analyzer.GetFileSet()
}
