# gomu Architecture

This document describes the architecture and design principles of gomu, a high-performance mutation testing tool for Go.

## Overview

gomu is designed with modularity, performance, and Go-specific optimizations in mind. The architecture consists of several loosely-coupled components that work together to provide efficient mutation testing.

## Core Design Principles

### 1. Modular Architecture
- Clear separation of concerns between components
- Each component has a single responsibility
- Easy to test and maintain individual modules

### 2. Performance First
- Parallel execution using goroutines
- Incremental analysis to avoid redundant work
- Efficient AST processing and mutation generation

### 3. Go-Specific Optimizations
- Leverage Go's type system for intelligent mutations
- Native Go toolchain integration
- Optimized for Go project structures and conventions

### 4. Developer Experience
- Transparent configuration and reporting
- Clear error messages and verbose output
- Easy integration with existing Go workflows

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   cmd/gomu      │  │   cobra CLI     │  │  config      │ │
│  │   main.go       │  │   framework     │  │  management  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                     Engine Layer                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                   pkg/gomu                              │ │
│  │             Main Mutation Engine                       │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                   Core Components                          │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐         │
│  │   Analysis   │ │   Mutation   │ │  Execution   │         │
│  │              │ │              │ │              │         │
│  │ • File       │ │ • AST        │ │ • Parallel   │         │
│  │   Discovery  │ │   Parsing    │ │   Testing    │         │
│  │ • Git Diff   │ │ • Mutation   │ │ • Result     │         │
│  │ • AST Parse  │ │   Generation │ │   Collection │         │
│  └──────────────┘ └──────────────┘ └──────────────┘         │
│                                                             │
│  ┌──────────────┐ ┌──────────────┐                         │
│  │   History    │ │   Report     │                         │
│  │              │ │              │                         │
│  │ • Incremental│ │ • Text       │                         │
│  │   Analysis   │ │ • JSON       │                         │
│  │ • JSON Store │ │ • HTML       │                         │
│  │ • Caching    │ │   (future)   │                         │
│  └──────────────┘ └──────────────┘                         │
└─────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. CLI Layer (`cmd/gomu`)

**Responsibility**: Command-line interface and user interaction

**Key Files**:
- `main.go`: Entry point with Cobra CLI setup
- Command definitions and flag parsing
- User input validation

**Design Decisions**:
- Uses Cobra framework for robust CLI experience
- Separates CLI logic from core business logic
- Provides both simple and advanced usage patterns

### 2. Engine Layer (`pkg/gomu`)

**Responsibility**: Main orchestration and coordination

**Key Files**:
- `engine.go`: Main mutation testing engine

**Key Functions**:
- Coordinates all core components
- Manages execution flow
- Handles high-level error handling and logging

### 3. Configuration (`internal/ci`)

**Responsibility**: CI environment detection and configuration

**Key Features**:
- Environment variable-based configuration
- CI/CD integration settings
- Quality gate configuration
- GitHub Actions integration

**Benefits over existing tools**:
- No configuration files required
- Command-line flag based configuration
- Environment auto-detection

### 4. Analysis (`internal/analysis`)

**Responsibility**: File discovery and AST parsing

**Key Features**:
- Go source file discovery
- Git integration for change detection
- AST parsing and position tracking
- File hashing for incremental analysis

**Performance Optimizations**:
- Efficient file walking with early termination
- Git diff integration for incremental analysis
- Parallel AST parsing (future)

### 5. Mutation Engine (`internal/mutation`)

**Responsibility**: Mutation generation and management

**Components**:
- `engine.go`: Main mutation coordination
- `arithmetic.go`: Arithmetic operator mutations
- `conditional.go`: Conditional operator mutations  
- `logical.go`: Logical operator mutations

**Mutation Strategy**:
- AST-based mutations (not text-based)
- Type-aware mutation generation
- Configurable mutation types
- Mutation limit enforcement

**Extensibility**:
- Plugin-like mutator interface
- Easy to add new mutation types
- Go-specific mutations (future)

### 6. Execution Engine (`internal/execution`)

**Responsibility**: Test execution and result collection

**Key Features**:
- Parallel test execution using goroutines
- Worker pool pattern for resource management
- Timeout handling and cancellation
- Result aggregation

**Performance Features**:
- Configurable worker count
- Early termination on test failure
- Efficient resource management

### 7. History Store (`internal/history`)

**Responsibility**: Incremental analysis and caching

**Key Features**:
- JSON-based history storage
- File hash tracking
- Result caching
- Statistics aggregation

**Advantages over PITest**:
- Transparent JSON format (vs opaque binary)
- Cross-platform compatibility
- Easy debugging and inspection
- Git-friendly format

### 8. Report Generator (`internal/report`)

**Responsibility**: Result formatting and output

**Output Formats**:
- Text: Human-readable console output
- JSON: Machine-readable for CI/CD
- HTML: Rich web-based reports (future)

**Report Features**:
- Mutation score calculation
- Detailed survived mutant listings
- Performance metrics
- Configurable verbosity

## Data Flow

### 1. Initialization Phase
```
User Input → Config Load → Engine Creation → Component Initialization
```

### 2. Discovery Phase
```
Path Input → File Discovery → Git Diff (optional) → Target File List
```

### 3. Analysis Phase
```
File List → AST Parsing → History Check → Filtered File List
```

### 4. Mutation Phase
```
Files → AST Analysis → Mutation Generation → Mutant List
```

### 5. Execution Phase
```
Mutants → Parallel Execution → Result Collection → Status Aggregation
```

### 6. Reporting Phase
```
Results → Statistics → Report Generation → Output
```

## Performance Considerations

### 1. Parallel Execution
- Worker pool pattern for controlled concurrency
- Goroutines for lightweight parallelism
- Configurable worker count based on system resources

### 2. Incremental Analysis
- File hashing to detect changes
- Git integration for change detection
- Result caching to avoid redundant work
- JSON-based storage for transparency

### 3. Memory Optimization
- Streaming AST processing
- Lazy loading of mutation data
- Efficient data structures
- Garbage collection optimization

### 4. I/O Optimization
- Efficient file discovery
- Batch operations where possible
- Minimal disk reads/writes
- Compressed history storage (future)

## Error Handling Strategy

### 1. Graceful Degradation
- Continue processing other files on single file errors
- Provide warnings for non-critical issues
- Fail fast only on configuration or system errors

### 2. Clear Error Messages
- Contextual error information
- Actionable error messages
- Verbose mode for detailed debugging

### 3. Recovery Mechanisms
- Automatic fallback to full analysis if incremental fails
- Graceful handling of corrupted history files
- Retry mechanisms for transient failures

## Extension Points

### 1. Mutator Interface
```go
type Mutator interface {
    Name() string
    CanMutate(node ast.Node) bool
    Mutate(node ast.Node, fset *token.FileSet) []Mutant
}
```

### 2. Reporter Interface (Future)
```go
type Reporter interface {
    Generate(summary *Summary) error
    Format() string
}
```

### 3. History Store Interface (Future)
```go
type Store interface {
    Load() error
    Save() error
    GetEntry(filePath string) (Entry, bool)
    UpdateFile(filePath string, mutants []Mutant, results []Result)
}
```

## Comparison with Existing Tools

### vs Gremlins
- **Performance**: Better parallel execution and incremental analysis
- **Configuration**: JSON vs YAML, more transparent defaults
- **Scalability**: Designed for large codebases from the start

### vs go-mutesting
- **Architecture**: More modular and maintainable design
- **Performance**: Better parallelization and Git integration
- **Features**: Incremental analysis and richer reporting

### vs PITest (Java)
- **Language Integration**: Native Go toolchain integration
- **History Format**: JSON vs opaque binary format
- **Cross-platform**: Better cross-platform compatibility

## Future Architecture Enhancements

### 1. Plugin System
- External mutator plugins
- Custom reporter plugins
- Language-specific extensions

### 2. Distributed Execution
- Remote worker nodes
- Cloud-based execution
- Kubernetes integration

### 3. Advanced Analysis
- Coverage-guided mutation
- AI-powered mutation prioritization
- Semantic mutation analysis

### 4. Integration Layer
- IDE extensions (VS Code, GoLand)
- CI/CD platform plugins
- Git hooks integration