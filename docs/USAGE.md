# Usage Guide

This guide provides detailed information on how to use gomu effectively for mutation testing in your Go projects.

## Installation

### Option 1: Install from source (recommended for development)
```bash
git clone https://github.com/sivchari/gomu.git
cd gomu
go build -o gomu ./cmd/gomu
```

### Option 2: Go install (when published)
```bash
go install github.com/sivchari/gomu/cmd/gomu@latest
```

## Basic Usage

### Run mutation testing on current directory
```bash
gomu run
```

### Run on specific directory
```bash
gomu run ./pkg/mypackage
```

### Enable verbose output
```bash
gomu run -v
```

### Use custom configuration file
```bash
gomu run --config custom-config.yaml
```

## Configuration

### Configuration File Locations

gomu looks for configuration files in the following order:
1. File specified with `--config` flag
2. `.gomu.yaml` in current directory
3. `gomu.yaml` in current directory  
4. `.gomu/config.yaml` in current directory
5. Default configuration if no file found

### Basic Configuration

Create a `.gomu.yaml` file in your project root:

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

# Mutation configuration
mutation:
  types:
    - "arithmetic"
    - "conditional" 
    - "logical"
  limit: 1000

# Incremental analysis
incremental:
  enabled: true
  historyFile: ".gomu_history.json"
  useGitDiff: true
  baseBranch: "main"

# Output configuration
output:
  format: "text"
  file: ""
```

### Configuration Reference

#### General Settings

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `verbose` | boolean | `false` | Enable detailed logging output |
| `workers` | integer | `4` | Number of parallel workers for test execution |

#### Test Settings

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `test_command` | string | `"go test"` | Command to run tests |
| `test_timeout` | integer | `30` | Test timeout in seconds |
| `test_patterns` | array | `["*_test.go"]` | Glob patterns for test files |
| `exclude_files` | array | `["vendor/", ".git/"]` | Files/directories to exclude |

#### Mutation Settings

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mutators` | array | `["arithmetic", "conditional", "logical"]` | Types of mutations to apply |
| `mutation_limit` | integer | `1000` | Maximum number of mutations per run |

#### Incremental Analysis

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `history_file` | string | `".gomu_history.json"` | File to store mutation history |
| `use_git_diff` | boolean | `true` | Use git diff for incremental analysis |
| `base_branch` | string | `"main"` | Base branch for git diff comparison |

#### Output Settings

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `output_format` | string | `"text"` | Output format: `text`, `json`, or `html` |
| `output_file` | string | `""` | File to write output (empty = stdout) |

## Advanced Configuration Examples

### High Performance Setup
```json
{
  "workers": 8,
  "mutation_limit": 5000,
  "test_timeout": 60,
  "use_git_diff": true,
  "verbose": false
}
```

### Development Setup
```json
{
  "workers": 2,
  "mutation_limit": 100,
  "test_timeout": 10,
  "use_git_diff": true,
  "verbose": true,
  "output_format": "text"
}
```

### CI/CD Setup
```json
{
  "workers": 4,
  "mutation_limit": 2000,
  "test_timeout": 120,
  "use_git_diff": false,
  "verbose": false,
  "output_format": "json",
  "output_file": "mutation-report.json"
}
```

### Microservice Setup
```json
{
  "workers": 2,
  "mutation_limit": 500,
  "test_timeout": 15,
  "exclude_files": ["vendor/", ".git/", "proto/", "docs/"],
  "mutators": ["arithmetic", "conditional"],
  "use_git_diff": true
}
```

## Understanding Output

### Text Format Output

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
```

### Mutation Status Meanings

- **Killed**: Mutation was detected by tests (good!)
- **Survived**: Mutation was not detected by tests (needs attention)
- **Timed Out**: Tests took too long to run
- **Error**: Build or runtime error occurred
- **Not Covered**: Mutation not covered by any tests

### Mutation Score Interpretation

- **90-100%**: Excellent test coverage
- **80-89%**: Good test coverage
- **70-79%**: Acceptable test coverage
- **Below 70%**: Poor test coverage, needs improvement

## Incremental Analysis

### How it Works

1. **File Hashing**: gomu calculates hashes for source files and tests
2. **Git Integration**: Detects changed files using `git diff`
3. **Result Caching**: Reuses previous results for unchanged code
4. **Smart Execution**: Only runs mutations on modified code

### Benefits

- **Speed**: Can reduce execution time from minutes to seconds
- **Efficiency**: Focuses testing effort on changed code
- **Developer Friendly**: Fast feedback loop for development

### Setup for Maximum Benefit

1. **Enable Git Integration**:
```json
{
  "use_git_diff": true,
  "base_branch": "main"
}
```

2. **Use in Development Workflow**:
```bash
# Make changes to code
git add .
gomu run  # Only tests changed files
```

3. **History File Management**:
- Add `.gomu_history.json` to `.gitignore` for local development
- Or commit it for team-wide incremental analysis

## Workflow Integration

### Pre-commit Hook

Create `.git/hooks/pre-commit`:
```bash
#!/bin/bash
gomu run --config .gomu-precommit.json
if [ $? -ne 0 ]; then
    echo "Mutation testing failed"
    exit 1
fi
```

### CI/CD Integration

#### GitHub Actions
```yaml
- name: Run mutation testing
  run: |
    go build -o gomu ./cmd/gomu
    ./gomu run --config .gomu.yaml
```

#### GitLab CI
```yaml
mutation_testing:
  script:
    - go build -o gomu ./cmd/gomu
    - ./gomu run --config .gomu.yaml
```

### Makefile Integration

```makefile
.PHONY: mutation-test
mutation-test:
	@echo "Running mutation testing..."
	@./gomu run

.PHONY: mutation-test-ci
mutation-test-ci:
	@echo "Running mutation testing for CI..."
	@./gomu run --config .gomu.yaml
```

## Troubleshooting

### Common Issues

#### No mutations generated
- Check that target files contain mutatable code
- Verify `exclude_files` configuration
- Ensure files are not filtered out by patterns

#### Tests taking too long
- Increase `test_timeout` value
- Reduce `workers` count for less parallel load
- Check for infinite loops in test code

#### Git diff not working
- Ensure you're in a git repository
- Check that `base_branch` exists
- Verify git is accessible from command line

#### High memory usage
- Reduce `workers` count
- Set lower `mutation_limit`
- Exclude large vendor directories

### Performance Tuning

#### For Large Codebases
```json
{
  "workers": 6,
  "mutation_limit": 2000,
  "use_git_diff": true,
  "exclude_files": ["vendor/", "generated/", "mocks/"]
}
```

#### For Small Projects
```json
{
  "workers": 2,
  "mutation_limit": 500,
  "test_timeout": 15
}
```

#### For CI Environments
```json
{
  "workers": 4,
  "test_timeout": 60,
  "use_git_diff": false,
  "verbose": false
}
```

## Best Practices

### 1. Start Small
- Begin with a single package or module
- Gradually expand to larger codebases
- Use `mutation_limit` to control scope

### 2. Incremental Adoption
- Enable incremental analysis for development
- Use full analysis for CI/CD
- Focus on critical code paths first

### 3. Configuration Management
- Use different configs for dev/CI environments
- Version control your configuration files
- Document configuration choices for team

### 4. Interpreting Results
- Focus on high-value survived mutants
- Don't aim for 100% mutation score blindly
- Consider the cost/benefit of additional tests

### 5. Team Integration
- Set mutation score thresholds for CI
- Review survived mutants in code reviews
- Use mutation testing to guide test writing

## Examples

See the `examples/` directory for:
- Basic calculator with comprehensive tests
- Example configuration files
- Integration examples

## Getting Help

If you encounter issues or have questions:
1. Check this usage guide
2. Review the [Architecture documentation](ARCHITECTURE.md)
3. Look at existing examples
4. Open an issue on GitHub