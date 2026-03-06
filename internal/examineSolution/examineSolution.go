package examineSolution

import "math"

func ErrorNorm(u []float64, N int, solution func(float64) float64, uzel []float64) float64 {
	norma := 0.
	res := make([]float64, N+1)
	for i := 0; i < N+1; i++ {
		res[i] = math.Abs(solution(uzel[i]) - u[i])
		if res[i] > norma {
			norma = res[i]
		}
	}
	return norma
}
