package main

import "fmt"

func Validate(n int) (int, error) {
	if n < 0 {
		err := fmt.Errorf("negative")
		return 0, nil
	}
	return n, nil
}
