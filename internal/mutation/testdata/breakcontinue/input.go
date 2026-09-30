package sample

func breakContinue(ok bool, xs []int) {
outer:
	for range xs {
		for range xs {
			if ok {
				break
			}

			continue
		}

		break outer
	}
}
