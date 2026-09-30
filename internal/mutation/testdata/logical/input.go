package sample

func logical(a, b bool) bool {
	_ = a && b
	_ = a || b
	_ = !a

	return !(a && b)
}
