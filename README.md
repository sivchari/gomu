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
- **Interface Mutations**: Targeted interface implementation testing

### 🔧 CI/CD Integration
- **Quality Gates**: Configurable mutation score thresholds
- **GitHub Integration**: Automatic PR comments with test results
- **Multiple Output Formats**: JSON, HTML, and console reporting
- **Artifact Generation**: CI-friendly report artifacts

### 🛠️ Developer Experience
- **YAML Configuration**: Clean, unified configuration format
- **Rich Reporting**: Detailed HTML, JSON, and console output formats
- **CLI Integration**: Simple command-line interface with intuitive subcommands
- **Flexible Targeting**: Run on specific files, directories, or changed files only

### 📊 Advanced Analysis
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

1. **Initialize configuration:**
```bash
gomu config init
```

2. **Run on current directory:**
```bash
gomu run
```

3. **Run in CI environment:**
```bash
gomu run --ci-mode
```

4. **Run on specific directory:**
```bash
gomu run ./pkg/mypackage
```

5. **Verbose output:**
```bash
gomu run -v
```

## Configuration

Create a `.gomu.yaml` file in your project root using:

```bash
gomu config init
```

Example unified configuration (works for both local and CI environments):

```yaml
# General settings
verbose: false
workers: 4

# Test configuration
test:
  command: "go test"
  timeout: 30
  patterns:
    - "*_test.go"
  exclude:
    - "vendor/"
    - ".git/"
    - "node_modules/"

# Mutation configuration
mutation:
  types:
    - "arithmetic"
    - "conditional" 
    - "logical"
  limit: 1000

# Incremental analysis for performance
incremental:
  enabled: true
  historyFile: ".gomu_history.json"
  useGitDiff: true
  baseBranch: "main"

# Output configuration
output:
  format: "json"
  file: ""
  html:
    template: ""
    css: ""

# CI/CD integration - single config file for both local and CI environments
ci:
  enabled: true
  
  # Environment detection (auto-detects CI vs local)
  mode: "auto"  # auto, local, ci
  
  # Quality gates for CI/CD
  qualityGate:
    enabled: true
    minMutationScore: 80.0
    maxSurvivors: 0
    failOnQualityGate: true
    gradualEnforcement: false
    baselineFile: ""
  
  # GitHub integration (only active in CI)
  github:
    enabled: true
    prComments: true
    badges: true
    token: "${GITHUB_TOKEN}"
    repository: "${GITHUB_REPOSITORY}"
    prNumber: "${GITHUB_PR_NUMBER}"
    baseRef: "${GITHUB_BASE_REF}"
    headRef: "${GITHUB_HEAD_REF}"
  
  # CI reports and artifacts
  reports:
    formats:
      - "json"
      - "html"
    outputDir: "."
    artifacts: true
    
  # CI-specific optimizations
  performance:
    parallelWorkers: 4
    timeoutMultiplier: 1.5
    incrementalAnalysis: true
```

### Single Configuration Philosophy

gomu uses a **single configuration file** approach:

- **Local Development**: Run `gomu run` using the same `.gomu.yaml`
- **CI Environment**: Run `gomu run --ci-mode` using the same `.gomu.yaml`
- **Auto-Detection**: gomu automatically detects the environment and applies appropriate settings
- **Environment Variables**: CI-specific values like `GITHUB_TOKEN` are injected automatically

### Configuration Validation

Validate your configuration:

```bash
gomu config validate
```

## Commands

### Basic Usage

- `gomu run [path]` - Run mutation testing
- `gomu run --ci-mode [path]` - Run mutation testing in CI/CD environment
- `gomu version` - Show version information

### Configuration Management

- `gomu config init` - Initialize a new configuration file
- `gomu config validate [config-file]` - Validate configuration

### CI/CD Command Options

The `gomu run --ci-mode` command includes additional options for CI/CD environments:

```bash
gomu run --ci-mode --threshold 85.0                    # Set quality gate threshold
gomu run --ci-mode --format html                       # Output format (json, html, console)
gomu run --ci-mode --fail-on-gate=false               # Don't fail build on quality gate
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
        go-version: '1.24'
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
        go-version: '1.24'
    
    - name: Install gomu
      run: go install github.com/sivchari/gomu/cmd/gomu@latest
    
    - name: Run mutation testing
      run: gomu run --ci-mode
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        GITHUB_REPOSITORY: ${{ github.repository }}
        GITHUB_PR_NUMBER: ${{ github.event.number }}
        GITHUB_BASE_REF: ${{ github.event.pull_request.base.ref }}
        GITHUB_HEAD_REF: ${{ github.event.pull_request.head.ref }}
    
    - name: Upload mutation report
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: mutation-report
        path: |
          mutation-report.html
          mutation-report.json
```

### Quality Gates

Quality gates automatically fail the build when mutation score falls below threshold:

- Configurable minimum mutation score
- Optional maximum survivor count
- Gradual enforcement for legacy codebases
- Detailed failure reporting

## Example Output

### Console Output
```
🧬 Starting CI Mutation Testing...
📁 Working directory: .
⚙️  Configuration: .gomu.yaml

Analyzing files for changes...
Incremental Analysis Report
==========================
✓ src/calculator.go - File content changed
✓ src/utils.go - No previous history

Summary: 2 files need testing, 3 files skipped
Performance improvement: 60.0% files skipped

Running mutation testing on 2 files...
Quality Gate: PASSED (Score: 84.4%)

✅ CI mutation testing completed successfully
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
4. **JSON Storage**: Transparent, debuggable history format

This can reduce execution time from minutes to seconds on large codebases.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [PITest](https://pitest.org/) for Java
- Builds upon research in mutation testing
- Thanks to the Go community for excellent tooling
