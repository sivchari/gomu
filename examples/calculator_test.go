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

	if got := calc.Multiply(3, 4); got != 12 {
		t.Errorf("Multiply(3, 4) = %v, want 12", got)
	}

	if got := calc.Multiply(0, 5); got != 0 {
		t.Errorf("Multiply(0, 5) = %v, want 0", got)
	}
}

func TestCalculator_Divide(t *testing.T) {
	calc := &Calculator{}

	if got := calc.Divide(8, 2); got != 4 {
		t.Errorf("Divide(8, 2) = %v, want 4", got)
	}

	if got := calc.Divide(5, 0); got != 0 {
		t.Errorf("Divide(5, 0) = %v, want 0", got)
	}
}

func TestCalculator_IsPositive(t *testing.T) {
	calc := &Calculator{}

	if !calc.IsPositive(5) {
		t.Error("IsPositive(5) should return true")
	}

	if calc.IsPositive(-5) {
		t.Error("IsPositive(-5) should return false")
	}

	if calc.IsPositive(0) {
		t.Error("IsPositive(0) should return false")
	}
}

func TestCalculator_IsEven(t *testing.T) {
	calc := &Calculator{}

	if !calc.IsEven(4) {
		t.Error("IsEven(4) should return true")
	}

	if calc.IsEven(3) {
		t.Error("IsEven(3) should return false")
	}
}
