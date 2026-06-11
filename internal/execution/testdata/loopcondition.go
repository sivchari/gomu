package main

func SumTo(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		total = total + i
	}
	return total
}
