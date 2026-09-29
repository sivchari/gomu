package main

func Check(err error) error {
	switch err {
	case nil:
	default:
		return err
	}
	return nil
}
