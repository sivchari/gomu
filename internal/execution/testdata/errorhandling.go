package main

import "fmt"

func Validate(n int) (int, error) {
	if n < 0 {
		err := fmt.Errorf("negative")
		return 0, err
	}
	return n, nil
}
