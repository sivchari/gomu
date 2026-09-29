package execution

import (
	"reflect"
	"testing"
)

func TestSplitBuildFlags(t *testing.T) {
	tests := []struct {
		name     string
		testArgs []string
		want     []string
	}{
		{
			name:     "no args",
			testArgs: nil,
			want:     nil,
		},
		{
			name:     "test-only flag alone",
			testArgs: []string{"-short"},
			want:     nil,
		},
		{
			name:     "test-only flags with separate run value",
			testArgs: []string{"-short", "-race", "-run", "^TestUnit"},
			want:     []string{"-race"},
		},
		{
			name:     "tags with equals form",
			testArgs: []string{"-tags=integration"},
			want:     []string{"-tags=integration"},
		},
		{
			name:     "tags with separate value form",
			testArgs: []string{"-tags", "integration"},
			want:     []string{"-tags", "integration"},
		},
		{
			name:     "double dash boolean flag",
			testArgs: []string{"--race"},
			want:     []string{"--race"},
		},
		{
			name:     "double dash value flag with equals",
			testArgs: []string{"--tags=integration"},
			want:     []string{"--tags=integration"},
		},
		{
			name:     "value flag at end of slice with no following value",
			testArgs: []string{"-tags"},
			want:     []string{"-tags"},
		},
		{
			name:     "mixed build and test flags preserve build order",
			testArgs: []string{"-v", "-tags=integration", "-race", "-count=1"},
			want:     []string{"-v", "-tags=integration", "-race"},
		},
		{
			name:     "non-flag positional value between flags is ignored",
			testArgs: []string{"-run", "TestFoo", "-msan"},
			want:     []string{"-msan"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitBuildFlags(tt.testArgs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitBuildFlags(%v) = %v, want %v", tt.testArgs, got, tt.want)
			}
		})
	}
}

func TestParseFlagName(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantName string
		wantHas  bool
	}{
		{"single dash bool", "-race", "race", false},
		{"double dash bool", "--race", "race", false},
		{"single dash with value", "-tags=integration", "tags", true},
		{"double dash with value", "--tags=integration", "tags", true},
		{"not a flag", "^TestUnit", "", false},
		{"bare dash", "-", "", false},
		{"triple dash", "---race", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, hasValue := parseFlagName(tt.arg)
			if name != tt.wantName || hasValue != tt.wantHas {
				t.Errorf("parseFlagName(%q) = (%q, %v), want (%q, %v)", tt.arg, name, hasValue, tt.wantName, tt.wantHas)
			}
		})
	}
}
