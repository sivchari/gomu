# gomu

A high-performance mutation testing tool for Go that helps validate the quality of your test suite.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What is Mutation Testing?

Mutation testing evaluates the quality of your test suite by introducing controlled changes (mutations) to your code and checking if your tests catch them. If a test fails when code is mutated, the mutation is "killed" (good). If tests still pass, the mutation "survived" (indicates weak tests).

## Features

### High Performance
- **Incremental Analysis**: Only test changed files with Git integration
- **Parallel Execution**: Leverage goroutines for concurrent mutation testing
- **Efficient AST Processing**: Fast Go code analysis and mutation generation

### Go-Specific Optimizations
- **Type-Safe Mutations**: Leverage Go's type system for intelligent mutations
- **Error Handling Patterns**: Specialized mutations for Go error handling
- **Interface Mutations**: Targeted interface implementation testing

### CI/CD Integration
- **Quality Gates**: Configurable mutation score thresholds
- **GitHub Integration**: Automatic PR comments with test results
- **Multiple Output Formats**: JSON, HTML, and console reporting
- **Artifact Generation**: CI-friendly report artifacts

### Developer Experience
- **CLI-Based Configuration**: Simple command-line flags for all settings
- **Rich Reporting**: Detailed HTML, JSON, and console output formats
- **Flexible Targeting**: Run on specific files, directories, or changed files only
- **.gomuignore Support**: Exclude files and directories from mutation testing

### Advanced Analysis
- **History Tracking**: JSON-based incremental analysis for faster reruns
- **Git Integration**: Automatic detection of changed files
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

2. **Run in CI environment:**
```bash
gomu run --ci-mode
```

3. **Run on specific directory:**
```bash
gomu run ./pkg/mypackage
```

4. **Verbose output:**
```bash
gomu run -v
```

5. **Custom threshold and workers:**
```bash
gomu run --threshold 85.0 --workers 8
```

## Commands

### Basic Usage

- `gomu run [path]` - Run mutation testing on the specified path (default: current directory)
- `gomu version` - Show version information

### Run Command Options

| Flag | Default | Description |
|------|---------|-------------|
| `--ci-mode` | `false` | Enable CI mode with quality gates and GitHub integration |
| `--threshold` | `80.0` | Minimum mutation score threshold (0-100) |
| `--workers` | `4` | Number of parallel workers |
| `--timeout` | `30` | Test timeout in seconds |
| `--incremental` | `true` | Enable incremental analysis |
| `--base-branch` | `main` | Base branch for incremental analysis |
| `--output` | `json` | Output format (json, html, console) |
| `--fail-on-gate` | `true` | Fail build when quality gate is not met |
| `-v, --verbose` | `false` | Verbose output |

### Examples

```bash
# Basic run with default settings
gomu run

# Run with custom threshold
gomu run --threshold 85.0

# Run in CI mode with HTML output
gomu run --ci-mode --output html

# Run with more workers and longer timeout
gomu run --workers 8 --timeout 60

# Run on specific package with verbose output
gomu run ./internal/mypackage -v

# Disable incremental analysis
gomu run --incremental=false
```

## .gomuignore

Create a `.gomuignore` file in your project root to exclude files and directories from mutation testing. The syntax is similar to `.gitignore`:

```
# Exclude directories
cmd/
vendor/
testdata/

# Exclude specific files
*_generated.go

# Negate pattern (include previously excluded)
!cmd/important/
```

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

### Bitwise Mutations
- Replace `&` with `|`, `^`
- Replace `|` with `&`, `^`
- Replace `^` with `&`, `|`
- Replace `>>` with `<<`
- Replace `<<` with `>>`

## CI/CD Integration

### GitHub Actions

#### Using the GitHub Action (Recommended)

The easiest way to integrate gomu into your workflow is using the official GitHub Action:

```yaml
name: Mutation Testing

on:
  pull_request:
    branches: [main]

jobs:
  mutation-test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
      issues: write

    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Run mutation testing
      uses: sivchari/gomu@main
      with:
        go-version: '1.21'
        threshold: '80'
        workers: '4'
        timeout: '30'
        upload-artifacts: 'true'
        comment-pr: 'true'
```

#### Required Permissions

**Important**: For PR comments and artifact uploads to work, your workflow needs the following permissions:

```yaml
permissions:
  contents: read
  pull-requests: write
  issues: write
```

#### Available Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `go-version` | Go version to use | `1.21` |
| `version` | gomu version to use (latest, nightly, local, or specific version) | `latest` |
| `working-directory` | Working directory for the action | `.` |
| `threshold` | Minimum mutation score threshold (0-100) | `80` |
| `workers` | Number of parallel workers | `4` |
| `timeout` | Test timeout in seconds | `30` |
| `incremental` | Enable incremental analysis | `true` |
| `base-branch` | Base branch for incremental analysis | `main` |
| `output` | Output format (json, html, console) | `json` |
| `fail-on-gate` | Whether to fail the build if quality gate is not met | `true` |
| `upload-artifacts` | Whether to upload mutation reports as artifacts | `true` |
| `comment-pr` | Whether to comment on pull requests with results | `true` |

#### Outputs

| Output | Description |
|--------|-------------|
| `mutation-score` | The mutation score percentage |
| `total-mutants` | Total number of mutants generated |
| `killed-mutants` | Number of killed mutants |
| `survived-mutants` | Number of survived mutants |

#### Manual Setup

If you prefer to set up the workflow manually:

```yaml
name: Mutation Testing

on:
  pull_request:
    branches: [main]

jobs:
  mutation-test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
      issues: write

    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'

    - name: Install gomu
      run: go install github.com/sivchari/gomu/cmd/gomu@latest

    - name: Run mutation testing
      run: gomu run --ci-mode --threshold 80.0
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        GITHUB_REPOSITORY: ${{ github.repository }}
        GITHUB_PR_NUMBER: ${{ github.event.number }}
        GITHUB_BASE_REF: ${{ github.event.pull_request.base.ref }}
        GITHUB_HEAD_REF: ${{ github.event.pull_request.head.ref }}

    - name: Upload mutation report
      uses: actions/upload-artifact@v4
      if: always()
      with:
        name: mutation-report
        path: |
          mutation-report.html
          mutation-report.json
```

### Quality Gates

Quality gates automatically fail the build when mutation score falls below threshold:

- Configurable minimum mutation score via `--threshold`
- Fail or continue build via `--fail-on-gate`
- Detailed failure reporting in CI output

## Example Output

### Console Output
```
Running mutation testing with the following settings:
  Path: .
  CI Mode: true
  Workers: 4
  Timeout: 30 seconds
  Output: json
  Incremental: true
  Base Branch: main
  Threshold: 80.0%
  Fail on Gate: true

Analyzing files for changes...
Incremental Analysis Report
==========================
- src/calculator.go - File content changed
- src/utils.go - No previous history

Summary: 2 files need testing, 3 files skipped
Performance improvement: 60.0% files skipped

Running mutation testing on 2 files...
Quality Gate: PASSED (Score: 84.4%)

Mutation testing completed successfully
```

### HTML Report

The HTML report provides:
- Interactive mutation score dashboard
- File-by-file mutation breakdown
- Survived mutant details with code snippets
- Quality gate status and recommendations

## Incremental Analysis

gomu features PITest-inspired incremental analysis that dramatically speeds up repeated runs:

1. **File Hashing**: Tracks changes to source files and tests
2. **Git Integration**: Automatically detects changed files since last commit
3. **Result Caching**: Reuses previous results for unchanged code
4. **JSON Storage**: Transparent, debuggable history format (`.gomu_history.json`)

This can reduce execution time from minutes to seconds on large codebases.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [PITest](https://pitest.org/) for Java
- Builds upon research in mutation testing
- Thanks to the Go community for excellent tooling
