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

### Use custom settings
```bash
gomu run --workers=8 --timeout=60 --threshold=85
```

## Configuration

### Command Line Configuration

gomu uses command-line flags for all configuration. No configuration files are required.

### Command Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--workers` | int | `4` | Number of parallel workers |
| `--timeout` | int | `30` | Test timeout in seconds |
| `--threshold` | float64 | `80.0` | Minimum mutation score threshold |
| `--output` | string | `json` | Output format (json, html, console) |
| `--incremental` | bool | `true` | Enable incremental analysis |
| `--base-branch` | string | `main` | Base branch for incremental analysis |
| `--ci-mode` | bool | `false` | Enable CI mode with quality gates |
| `--fail-on-gate` | bool | `true` | Fail build when quality gate is not met |
| `-v, --verbose` | bool | `false` | Enable verbose output |

### Basic Usage Examples

```bash
# Default settings
gomu run

# High performance
gomu run --workers=8 --timeout=60

# Development mode
gomu run --workers=2 --verbose

# CI mode
gomu run --ci-mode --threshold=85
```

### Configuration Reference

#### General Settings

See command line flags table above for all available options.

## Advanced Configuration Examples

### High Performance Setup
```bash
gomu run --workers=8 --timeout=60 --incremental=true
```

### Development Setup
```bash
gomu run --workers=2 --timeout=10 --incremental=true --verbose
```

### CI/CD Setup
```bash
gomu run --ci-mode --workers=4 --timeout=120 --output=json
```

### Microservice Setup
```bash
gomu run --workers=2 --timeout=15 --incremental=true
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
```bash
gomu run --incremental=true --base-branch=main
```

2. **Use in Development Workflow**:
```bash
# Make changes to code
git add .
gomu run  # Only tests changed files (incremental is enabled by default)
```

3. **History File Management**:
- Add `.gomu_history.json` to `.gitignore` for local development
- Or commit it for team-wide incremental analysis

## Workflow Integration

### Pre-commit Hook

Create `.git/hooks/pre-commit`:
```bash
#!/bin/bash
gomu run --workers=2 --timeout=15 --threshold=80
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
    ./gomu run --ci-mode --threshold=80
```

#### GitLab CI
```yaml
mutation_testing:
  script:
    - go build -o gomu ./cmd/gomu
    - ./gomu run --ci-mode --threshold=80
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
	@./gomu run --ci-mode --threshold=80
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
```bash
gomu run --workers=6 --incremental=true
```

#### For Small Projects
```bash
gomu run --workers=2 --timeout=15
```

#### For CI Environments
```bash
gomu run --ci-mode --workers=4 --timeout=60 --incremental=false
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