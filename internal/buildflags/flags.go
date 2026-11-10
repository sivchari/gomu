// Package buildflags classifies the flags forwarded after "--" so that every
// stage of a run (file discovery, type analysis, compilation, test execution)
// sees the same Go build configuration.
package buildflags

import "strings"

// buildValueFlags are "go build" flags that take a value, either as
// "-flag=value" or as a separate "-flag value" argument.
var buildValueFlags = map[string]bool{
	"tags":          true,
	"mod":           true,
	"modfile":       true,
	"gcflags":       true,
	"ldflags":       true,
	"asmflags":      true,
	"gccgoflags":    true,
	"buildmode":     true,
	"compiler":      true,
	"installsuffix": true,
	"pgo":           true,
	"pkgdir":        true,
	"toolexec":      true,
	"covermode":     true,
	"coverpkg":      true,
	"p":             true,
}

// buildBoolFlags are "go build" flags that take no value.
var buildBoolFlags = map[string]bool{
	"race":       true,
	"msan":       true,
	"asan":       true,
	"cover":      true,
	"trimpath":   true,
	"linkshared": true,
	"modcacherw": true,
	"a":          true,
	"n":          true,
	"v":          true,
	"x":          true,
	"work":       true,
	"buildvcs":   true,
}

// discoveryValueFlags and discoveryBoolFlags are the "go build" flags that
// change which files belong to a package or how modules resolve, and that
// "go list" also accepts. Flags that only tune compilation (-gcflags, -ldflags,
// -cover and friends) or print commands (-n, -x) are left out.
var (
	discoveryValueFlags = map[string]bool{
		"tags":     true,
		"mod":      true,
		"modfile":  true,
		"compiler": true,
	}
	discoveryBoolFlags = map[string]bool{
		"race":       true,
		"msan":       true,
		"asan":       true,
		"modcacherw": true,
	}
)

// Build returns the subset of testArgs that are also valid "go build" flags,
// since go build rejects test-only flags such as -short or -run.
func Build(testArgs []string) []string {
	return filter(testArgs, buildValueFlags, buildBoolFlags)
}

// Discovery returns the subset of testArgs that decides which production files
// the active build includes (such as -tags or -race) and that "go list"
// accepts. Test-selection flags such as -run are dropped, so they never change
// which files can be mutated.
func Discovery(testArgs []string) []string {
	return filter(testArgs, discoveryValueFlags, discoveryBoolFlags)
}

// filter keeps the arguments naming a flag in valueFlags (with its value,
// inline or as the next argument) or boolFlags, preserving their order.
func filter(testArgs []string, valueFlags, boolFlags map[string]bool) []string {
	var kept []string

	for i := 0; i < len(testArgs); i++ {
		arg := testArgs[i]

		name, hasValue := parseFlagName(arg)
		if name == "" {
			continue
		}

		switch {
		case valueFlags[name]:
			kept = append(kept, arg)

			if !hasValue && i+1 < len(testArgs) {
				i++

				kept = append(kept, testArgs[i])
			}
		case boolFlags[name]:
			kept = append(kept, arg)
		}
	}

	return kept
}

// parseFlagName extracts the flag name (without leading dashes) from a
// "-flag", "--flag" or "-flag=value" argument, and reports whether it carries
// an inline value. It returns an empty name for anything that is not a flag.
func parseFlagName(arg string) (name string, hasValue bool) {
	if !strings.HasPrefix(arg, "-") {
		return "", false
	}

	trimmed := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return "", false
	}

	if idx := strings.Index(trimmed, "="); idx >= 0 {
		return trimmed[:idx], true
	}

	return trimmed, false
}
