package main

import "testing"

func TestCalculator_Add(t *testing.T) {
	calc := &Calculator{}

	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"positive numbers", 2, 3, 5},
		{"negative numbers", -2, -3, -5},
		{"mixed signs", -2, 3, 1},
		{"zero", 0, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Add(tt.a, tt.b); got != tt.want {
				t.Errorf("Add(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCalculator_Subtract(t *testing.T) {
	calc := &Calculator{}

	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"positive result", 5, 3, 2},
		{"negative result", 3, 5, -2},
		{"zero result", 5, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Subtract(tt.a, tt.b); got != tt.want {
				t.Errorf("Subtract(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCalculator_Multiply(t *testing.T) {
	calc := &Calculator{}

	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"positive numbers", 3, 4, 12},
		{"zero multiplication", 0, 5, 0},
		{"multiply by zero", 5, 0, 0},
		{"negative numbers", -3, -4, 12},
		{"mixed signs", -3, 4, -12},
		{"multiply by one", 7, 1, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Multiply(tt.a, tt.b); got != tt.want {
				t.Errorf("Multiply(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCalculator_Divide(t *testing.T) {
	calc := &Calculator{}

	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"normal division", 8, 2, 4},
		{"division by zero", 5, 0, 0},
		{"negative dividend", -8, 2, -4},
		{"negative divisor", 8, -2, -4},
		{"both negative", -8, -2, 4},
		{"zero dividend", 0, 5, 0},
		{"division with remainder", 7, 3, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.Divide(tt.a, tt.b); got != tt.want {
				t.Errorf("Divide(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCalculator_IsPositive(t *testing.T) {
	calc := &Calculator{}

	tests := []struct {
		name string
		n    int
		want bool
	}{
		{"positive number", 5, true},
		{"negative number", -5, false},
		{"zero", 0, false},
		{"large positive", 1000, true},
		{"large negative", -1000, false},
		{"one", 1, true},
		{"negative one", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.IsPositive(tt.n); got != tt.want {
				t.Errorf("IsPositive(%v) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestCalculator_IsEven(t *testing.T) {
	calc := &Calculator{}

	tests := []struct {
		name string
		n    int
		want bool
	}{
		{"even number", 4, true},
		{"odd number", 3, false},
		{"zero", 0, true},
		{"negative even", -4, true},
		{"negative odd", -3, false},
		{"two", 2, true},
		{"one", 1, false},
		{"large even", 100, true},
		{"large odd", 101, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calc.IsEven(tt.n); got != tt.want {
				t.Errorf("IsEven(%v) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

// Additional edge case tests for better mutation coverage.
func TestCalculator_EdgeCases(t *testing.T) {
	calc := &Calculator{}

	// Test boundary conditions for divide by zero
	t.Run("divide by zero special cases", func(t *testing.T) {
		if got := calc.Divide(0, 0); got != 0 {
			t.Errorf("Divide(0, 0) = %v, want 0", got)
		}

		if got := calc.Divide(-1, 0); got != 0 {
			t.Errorf("Divide(-1, 0) = %v, want 0", got)
		}
	})

	// Test modulo operation in IsEven
	t.Run("modulo boundary cases", func(t *testing.T) {
		// Test values around boundaries
		if !calc.IsEven(2) {
			t.Error("IsEven(2) should return true")
		}

		if calc.IsEven(3) {
			t.Error("IsEven(3) should return false")
		}
		// Test that modulo operator works correctly
		if calc.IsEven(5) {
			t.Error("IsEven(5) should return false")
		}

		if !calc.IsEven(6) {
			t.Error("IsEven(6) should return true")
		}
	})

	// Test comparison operator in IsPositive
	t.Run("comparison boundary cases", func(t *testing.T) {
		// Test values around zero boundary
		if !calc.IsPositive(1) {
			t.Error("IsPositive(1) should return true")
		}

		if calc.IsPositive(0) {
			t.Error("IsPositive(0) should return false")
		}

		if calc.IsPositive(-1) {
			t.Error("IsPositive(-1) should return false")
		}
	})
}
