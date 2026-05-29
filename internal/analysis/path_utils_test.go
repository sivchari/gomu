package analysis

import "testing"

func TestIsExcludedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "vendor directory",
			path: "vendor/external.go",
			want: true,
		},
		{
			name: "testdata directory",
			path: "testdata/fixture.go",
			want: true,
		},
		{
			name: "nested testdata directory",
			path: "internal/execution/testdata/arithmetic.go",
			want: true,
		},
		{
			name: "nested vendor directory",
			path: "internal/vendor/pkg/lib.go",
			want: true,
		},
		{
			name: "vendor as exact directory segment",
			path: "vendor",
			want: true,
		},
		{
			name: "regular source file",
			path: "internal/mutation/engine.go",
			want: false,
		},
		{
			name: "substring vendor is not excluded",
			path: "internal/myvendor/lib.go",
			want: false,
		},
		{
			name: "substring testdata is not excluded",
			path: "internal/testdatautil/helper.go",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsExcludedPath(tt.path); got != tt.want {
				t.Errorf("IsExcludedPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
