package pingpong

func botAxis(ball, paddle float64) float64 {
	if ball > paddle+.04 {
		return 1
	}
	if ball < paddle-.04 {
		return -1
	}
	return 0
}
