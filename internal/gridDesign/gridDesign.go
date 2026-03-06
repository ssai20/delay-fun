package gridDesign

import "math"

func UniformMesh(N int) []float64 {
	h := make([]float64, N+1)

	for i := 0; i < N+1; i++ {
		h[i] = 1. / float64(N)
	}
	return h
}

func ShishkinMesh(epsilon float64, N int) []float64 {
	h := make([]float64, N+1)
	sigma := math.Min(0.5, 4.*epsilon*math.Log(float64(N)))

	for i := 0; i < N+1; i++ {
		if (i >= 0.) && (i <= N/2.) {
			h[i] = 2. * sigma / float64(N)
		}
		if (i > N/2.) && (i <= N) {
			h[i] = 2. * (1. - sigma) / float64(N)
		}
	}
	return h
}

func BakhvalovaMesh(epsilon float64, N int) []float64 {
	h := make([]float64, N+1)

	if epsilon <= math.Exp(-1) {
		sigma := math.Min(0.5, -4*epsilon*math.Log(epsilon))

		if sigma == 0.5 {
			// Равномерная сетка
			step := 1.0 / float64(N)
			for i := 0; i <= N; i++ {
				h[i] = step
			}
			return h
		}

		nFloat := float64(N)

		// Первая половина - экспоненциальное сгущение
		for i := 1; i <= N/2; i++ {
			termI := 1.0 - 2.0*(1.0-epsilon)*float64(i)/nFloat
			termIMinus1 := 1.0 - 2.0*(1.0-epsilon)*float64(i-1)/nFloat

			if termI <= 0 {
				termI = 1e-15
			}
			if termIMinus1 <= 0 {
				termIMinus1 = 1e-15
			}

			h[i-1] = -4.0 * epsilon * (math.Log(termI) - math.Log(termIMinus1))
		}

		// Вторая половина - линейная часть
		linearStep := 2.0 * (1.0 - sigma) / nFloat
		for i := N / 2; i <= N; i++ {
			h[i] = linearStep
		}

		// Коррекция центрального шага
		sumLeft := 0.0
		for i := 0; i < N/2-1; i++ {
			sumLeft += h[i]
		}
		h[N/2-1] = sigma - sumLeft

		// Гарантия положительности шагов
		minStep := epsilon * 1e-10
		for i := 0; i <= N; i++ {
			if h[i] < minStep {
				h[i] = minStep
			}
		}
	} else {
		// epsilon > exp(-1) - равномерная сетка
		step := 1.0 / float64(N)
		for i := 0; i <= N; i++ {
			h[i] = step
		}
	}

	return h
}
