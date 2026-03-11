package differenceScheme

import "math"

type ABCF struct {
	A, B, C, F []float64
}

func ClassicTeylorFormulasScheme(
	N int,
	epsilon float64,
	h []float64,
	delta float64,
	fn func(float64) float64,
	uzel []float64) ABCF {
	T := epsilon - delta*delta/2.

	A := make([]float64, N+1)
	B := make([]float64, N+1)
	C := make([]float64, N+1)
	F := make([]float64, N+1)

	for i := 1; i < N; i++ {
		A[i] = 2. * T / (h[i] + h[i+1]) / h[i]
		B[i] = -(2.*T/h[i]/h[i+1] + (delta+1.)/h[i+1] + 1.)
		C[i] = 2.*T/(h[i]+h[i+1])/h[i+1] + (1.+delta)/h[i+1]
		F[i] = fn(uzel[i])
	}

	return ABCF{
		A,
		B,
		C,
		F,
	}
}
func ModifiedTeylorFormulasScheme(
	N int,
	epsilon float64,
	h []float64,
	delta float64,
	fn func(float64) float64,
	uzel []float64,
	phi []float64,
	phiDelta []float64) ABCF {

	A := make([]float64, N+1)
	B := make([]float64, N+1)
	C := make([]float64, N+1)
	F := make([]float64, N+1)

	for i := 1; i < N; i++ {
		// Защита от деления на ноль
		denomR := phi[i+1] - phi[i]
		if denomR == 0 {
			denomR = 1e-300
		}

		R := epsilon*epsilon*(math.Exp(delta/epsilon)-1) - delta*epsilon //(phiDelta[i] - phi[i]) * h[i+1] / denomR

		// Ограничиваем R
		if R > 1e100 {
			R = 1e100
		}
		if R < -1e100 {
			R = -1e100
		}

		T := epsilon - R

		hSum := h[i] + h[i+1]
		if hSum < 1e-15 {
			hSum = 1e-15
		}

		A[i] = 2. * T / hSum / h[i]
		B[i] = -(2.*T/h[i]/h[i+1] + (delta+1.)/h[i+1] + 1.)
		C[i] = 2.*T/(hSum*h[i+1]) + (1.+delta)/h[i+1]
		F[i] = fn(uzel[i])
	}

	return ABCF{A, B, C, F}
}
