package sumpositive

func SumPositive(xs []int) int {
	total := 0
	for _, v := range xs {
		if v > 0 {
			total += v
		}
	}
	return total
}
