package main

func Count(items []int) int {
	count := 0
	for range items {
		count++
		continue
	}
	return count
}
