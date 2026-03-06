package thomasMethod

import "math"

func Progonka(A []float64, B []float64, C []float64, F []float64, N int, epsilon float64) []float64 {
	alpha := make([]float64, N+1)
	beta := make([]float64, N+1)
	c := make([]float64, N+1)

	alpha[1] = 0.
	beta[1] = 2.

	for i := 1; i < N; i++ {
		alpha[i+1] = -C[i] / (B[i] + alpha[i]*A[i])
		beta[i+1] = (-A[i]*beta[i] + F[i]) / (B[i] + alpha[i]*A[i])
	}

	c[N] = math.Exp(-1. / epsilon)

	for i := N - 1; i >= 0; i-- {
		c[i] = alpha[i+1]*c[i+1] + beta[i+1]
	}
	return c
}
