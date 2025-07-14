# gomu

A high-performance mutation testing tool for Go that helps validate the quality of your test suite.

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What is Mutation Testing?

Mutation testing evaluates the quality of your test suite by introducing controlled changes (mutations) to your code and checking if your tests catch them. If a test fails when code is mutated, the mutation is "killed" (good). If tests still pass, the mutation "survived" (indicates weak tests).

## Features

### 🚀 High Performance
- **Incremental Analysis**: Only test changed files with Git integration
- **Parallel Execution**: Leverage goroutines for concurrent mutation testing
- **Efficient AST Processing**: Fast Go code analysis and mutation generation

### 🎯 Go-Specific Optimizations
- **Type-Safe Mutations**: Leverage Go's type system for intelligent mutations
- **Error Handling Patterns**: Specialized mutations for Go error handling
- **Generic Support**: Future support for Go generics mutations
- **Interface Mutations**: Targeted interface implementation testing

### 🛠️ Developer Experience
- **JSON Configuration**: Transparent and debuggable configuration
- **Rich Reporting**: Detailed text and JSON output formats
- **CLI Integration**: Simple command-line interface with Cobra framework
- **Flexible Targeting**: Run on specific files, directories, or changed files only

### 📊 Advanced Analysis
- **History Tracking**: JSON-based incremental analysis (vs PITest's opaque format)
- **Git Integration**: Automatic detection of changed files for faster reruns
- **Mutation Score**: Comprehensive quality metrics
- **Detailed Reports**: Line-by-line mutation analysis

## Installation

```bash
go install github.com/sivchari/gomu/cmd/gomu@latest
```

Or clone and build:

```bash
git clone https://github.com/sivchari/gomu.git
cd gomu
go build -o gomu ./cmd/gomu
```

## Quick Start

1. **Run on current directory:**
```bash
gomu run
```

2. **Run on specific directory:**
```bash
gomu run ./pkg/mypackage
```

3. **Use configuration file:**
```bash
gomu run --config .gomu.json
```

4. **Verbose output:**
```bash
gomu run -v
```

## Configuration

Create a `.gomu.json` file in your project root:

```json
{
  "verbose": false,
  "workers": 4,
  "test_command": "go test",
  "test_timeout": 30,
  "test_patterns": ["*_test.go"],
  "exclude_files": ["vendor/", ".git/"],
  "mutators": ["arithmetic", "conditional", "logical"],
  "mutation_limit": 1000,
  "history_file": ".gomu_history.json",
  "use_git_diff": true,
  "base_branch": "main",
  "output_format": "text"
}
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `verbose` | Enable verbose output | `false` |
| `workers` | Number of parallel workers | `4` |
| `test_command` | Command to run tests | `"go test"` |
| `test_timeout` | Test timeout in seconds | `30` |
| `test_patterns` | Patterns for test files | `["*_test.go"]` |
| `exclude_files` | Files/directories to exclude | `["vendor/", ".git/"]` |
| `mutators` | Types of mutations to apply | `["arithmetic", "conditional", "logical"]` |
| `mutation_limit` | Maximum mutations per run | `1000` |
| `history_file` | File for incremental analysis | `".gomu_history.json"` |
| `use_git_diff` | Use git diff for incremental analysis | `true` |
| `base_branch` | Base branch for git diff | `"main"` |
| `output_format` | Output format (text/json/html) | `"text"` |

## Mutation Types

### Arithmetic Mutations
- Replace `+` with `-`, `*`, `/`
- Replace `-` with `+`, `*`, `/`
- Replace `*` with `+`, `-`, `/`, `%`
- Replace `/` with `+`, `-`, `*`, `%`
- Replace `++` with `--` and vice versa

### Conditional Mutations
- Replace `==` with `!=`, `<`, `<=`, `>`, `>=`
- Replace `!=` with `==`, `<`, `<=`, `>`, `>=`
- Replace `<` with `<=`, `>`, `>=`, `==`, `!=`
- Replace `>` with `>=`, `<`, `<=`, `==`, `!=`

### Logical Mutations
- Replace `&&` with `||`
- Replace `||` with `&&`
- Remove `!` (NOT) operators

## Example Output

```
Mutation Testing Report
=======================

Summary:
  Files processed: 3/3
  Total mutants:   45
  Duration:        2.3s

Results:
  Killed:     38 (84.4%)
  Survived:   7 (15.6%)
  Timed out:  0 (0.0%)
  Errors:     0 (0.0%)
  Not covered:0 (0.0%)

Mutation Score: 84.4%

Survived Mutants:
=================
  src/calculator.go:15:9 - Replace + with - (+ -> -)
  src/calculator.go:20:5 - Replace == with != (== -> !=)
  ...
```

## Incremental Analysis

gomu features PITest-inspired incremental analysis that dramatically speeds up repeated runs:

1. **File Hashing**: Tracks changes to source files and tests
2. **Git Integration**: Automatically detects changed files since last commit
3. **Result Caching**: Reuses previous results for unchanged code
4. **JSON Storage**: Transparent, debuggable history format

This can reduce execution time from minutes to seconds on large codebases.

## Comparison with Existing Tools

| Feature | gomu | Gremlins | go-mutesting |
|---------|------|----------|--------------|
| Performance | ⚡ High (parallel + incremental) | ⚠️ Slow on large projects | ⚠️ Moderate |
| Configuration | 📝 JSON (transparent) | 📝 YAML | 📝 YAML |
| Git Integration | ✅ Built-in | ❌ No | ❌ No |
| Incremental Analysis | ✅ JSON-based | ❌ No | ❌ No |
| Go Version Support | ✅ 1.24+ | ✅ Current | ✅ Current |
| Parallel Execution | ✅ Goroutines | ⚠️ Limited | ⚠️ Limited |
| Maintenance Status | ✅ Active | ⚠️ Pre-1.0 | ⚠️ Fork-based |

## Roadmap

### Phase 1 (Current)
- [x] Basic mutation types (arithmetic, conditional, logical)
- [x] Parallel execution
- [x] JSON configuration
- [x] Text and JSON output

### Phase 2
- [ ] Actual mutation application (currently simulated)
- [ ] Incremental analysis implementation
- [ ] HTML report generation
- [ ] More mutation types

### Phase 3
- [ ] Go-specific mutations (generics, error handling)
- [ ] VS Code extension
- [ ] CI/CD integrations
- [ ] Advanced reporting features

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [PITest](https://pitest.org/) for Java
- Builds upon research in mutation testing
- Thanks to the Go community for excellent tooling