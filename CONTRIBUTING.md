# Contributing to Gomu

Thank you for your interest in contributing to Gomu! This guide will help you get started with development.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Using the Scaffold Tool](#using-the-scaffold-tool)
- [Code Style](#code-style)
- [Testing](#testing)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Make (optional but recommended)

### Setting Up Your Development Environment

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/gomu.git
   cd gomu
   ```

3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/sivchari/gomu.git
   ```

4. Install dependencies:
   ```bash
   go mod download
   ```

5. Build the project:
   ```bash
   go build ./cmd/gomu
   ```

## Development Workflow

### Creating a New Branch

Always create a new branch for your changes:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### Making Changes

1. Make your changes in the appropriate files
2. Add tests for new functionality
3. Ensure all tests pass
4. Run linters and formatters

## Using the Scaffold Tool

Gomu provides a scaffold tool to help you quickly generate new mutator implementations. This is the recommended way to add new mutation operators.

### Building the Scaffold Tool

```bash
go build ./cmd/scaffold
```

### Generating a New Mutator

To create a new mutator, use the scaffold command:

```bash
./scaffold -mutator=<mutator_name>
```

For example, to create a "bitwise" mutator:

```bash
./scaffold -mutator=bitwise
```

This will generate:
- `internal/mutation/bitwise.go` - The mutator implementation
- `internal/mutation/bitwise_test.go` - Test file with basic test structure

### What the Scaffold Generates

The scaffold tool creates:

1. **Mutator Implementation** (`<name>.go`):
   - Basic mutator struct
   - `Name()` method
   - `CanMutate()` method (to be implemented)
   - `Mutate()` method (to be implemented)
   - Helper methods for specific mutation types

2. **Test File** (`<name>_test.go`):
   - Test for `Name()` method
   - Test structure for `CanMutate()`
   - Test structure for `Mutate()`
   - Example test cases to fill in

### Implementing Your Mutator

After scaffolding, you need to:

1. **Implement `CanMutate()`**: Determine if a given AST node can be mutated
   ```go
   func (m *YourMutator) CanMutate(node ast.Node) bool {
       // Check if this node type can be mutated
       // Return true if it can, false otherwise
   }
   ```

2. **Implement `Mutate()`**: Generate mutations for the node
   ```go
   func (m *YourMutator) Mutate(node ast.Node, fset *token.FileSet) []Mutant {
       // Generate and return mutations for the node
   }
   ```

3. **Register the Mutator**: Add your mutator to the engine in `internal/mutation/engine.go`:
   ```go
   func NewEngine() *Engine {
       return &Engine{
           mutators: []Mutator{
               // ... existing mutators
               &YourMutator{},
           },
       }
   }
   ```

4. **Write Tests**: Complete the test cases in the generated test file

### Example: Adding a String Mutator

```bash
# Generate the scaffold
./scaffold -mutator=string

# This creates:
# - internal/mutation/string.go
# - internal/mutation/string_test.go

# Now implement the mutator logic in string.go
# Add test cases in string_test.go
# Register in engine.go
```

## Code Style

### Go Code

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Use `golangci-lint` for linting:
  ```bash
  golangci-lint run
  ```

### File Organization

- Keep files focused and single-purpose
- Group related functionality in packages
- Use meaningful package and file names

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/mutation

# Run tests with verbose output
go test -v ./...
```

### Writing Tests

- Write table-driven tests when appropriate
- Include both positive and negative test cases
- Test edge cases
- Aim for high test coverage of critical paths
- Keep tests focused and avoid redundancy

Example test structure:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "basic case",
            input:    "input",
            expected: "output",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Feature(tt.input)
            if result != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

## Commit Guidelines

We follow conventional commit format for clear commit history.

### Commit Message Format

```
<type>: <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring
- `test`: Adding or modifying tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements
- `ci`: CI/CD changes

### Examples

```bash
# Feature
git commit -m "feat: add string mutation operator"

# Bug fix
git commit -m "fix: handle nil pointer in arithmetic mutator"

# Documentation
git commit -m "docs: update README with new mutation types"

# Test improvements
git commit -m "test: simplify redundant test cases in mutator_test.go"

# Multiple changes (with body)
git commit -m "refactor: consolidate mutation logic

- Extract common mutation patterns
- Reduce code duplication
- Improve error handling"
```

## Pull Request Process

1. **Update your branch**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Ensure quality**:
   - All tests pass
   - Code is formatted (`gofmt`)
   - Linter passes (`golangci-lint run`)
   - Documentation is updated if needed

3. **Create Pull Request**:
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changes you made and why
   - Include test results if relevant

4. **PR Description Template**:
   ```markdown
   ## Summary
   Brief description of changes

   ## Motivation
   Why these changes are needed

   ## Changes
   - Change 1
   - Change 2

   ## Testing
   How the changes were tested

   ## Checklist
   - [ ] Tests pass
   - [ ] Code is formatted
   - [ ] Documentation updated (if needed)
   ```

5. **Address Review Comments**:
   - Respond to all feedback
   - Make requested changes
   - Push updates to the same branch

## Questions and Support

If you have questions or need help:

1. Check existing issues and discussions
2. Open a new issue with the question label
3. Join discussions in pull requests

## License

By contributing to Gomu, you agree that your contributions will be licensed under the same license as the project (MIT License).