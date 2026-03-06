package gridDesign

func FindPoints(h []float64, N int) []float64 {
	uzel := make([]float64, N+1)
	uzel[0] = 0.
	for i := 1; i < N+1; i++ {
		uzel[i] = uzel[i-1] + h[i-1]
	}
	return uzel
}
