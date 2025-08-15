package main

// Export internal functions for testing
var (
	FindMutationDir  = &findMutationDir
	GenerateFile     = generateFile
	GenerateRegistry = generateRegistry
)

// Allow overriding for tests
var findMutationDir = findMutationDirImpl
