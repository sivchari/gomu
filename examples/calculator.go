// Package main demonstrates the use of gomu mutation testing on a simple calculator.
package main

import "fmt"

// Calculator provides basic arithmetic operations.
type Calculator struct{}

// Add returns the sum of two integers.
func (c *Calculator) Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers.
func (c *Calculator) Subtract(a, b int) int {
	return a - b
}

// Multiply returns the product of two integers.
func (c *Calculator) Multiply(a, b int) int {
	return a * b
}

// Divide returns the quotient of two integers.
func (c *Calculator) Divide(a, b int) int {
	if b == 0 {
		return 0
	}

	return a / b
}

// IsPositive checks if a number is positive.
func (c *Calculator) IsPositive(n int) bool {
	return n > 0
}

// IsEven checks if a number is even.
func (c *Calculator) IsEven(n int) bool {
	return n%2 == 0
}

func main() {
	calc := &Calculator{}

	fmt.Println("Calculator example")
	fmt.Println("2 + 3 =", calc.Add(2, 3))
	fmt.Println("5 - 2 =", calc.Subtract(5, 2))
	fmt.Println("3 * 4 =", calc.Multiply(3, 4))
	fmt.Println("8 / 2 =", calc.Divide(8, 2))
}
