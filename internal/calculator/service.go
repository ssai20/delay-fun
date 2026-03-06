package calculator

import (
	"delay-argument-go/internal/differenceScheme"
	"delay-argument-go/internal/examineSolution"
	"delay-argument-go/internal/gridDesign"
	"delay-argument-go/internal/latex"
	"delay-argument-go/internal/models"
	"delay-argument-go/internal/thomasMethod"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Job struct {
	ID         string
	Request    models.CalculationRequest
	Status     string
	PDFPath    string
	Error      string
	Classic    [][]string
	Modified   [][]string
	mu         sync.RWMutex
	ResultsDir string
}

func NewJob(req models.CalculationRequest, resultsDir string) *Job {
	return &Job{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		Request:    req,
		Status:     "pending",
		ResultsDir: resultsDir,
	}
}

func RunJob(job *Job) {
	job.mu.Lock()
	job.Status = "processing"
	job.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			job.mu.Lock()
			job.Status = "failed"
			job.Error = fmt.Sprintf("Panic: %v", r)
			job.mu.Unlock()
		}
	}()

	classic := make([][]string, 9)
	modified := make([][]string, 9)

	i := 0
	for e := job.Request.EpsilonStart; e >= job.Request.EpsilonMin; e /= 10. {
		classic[i] = make([]string, 5)
		modified[i] = make([]string, 5)
		j := 0

		for n := job.Request.NStart; n <= job.Request.NMax; n *= 2 {
			d := job.Request.Delta

			solution := func(x float64) float64 {
				return math.Cos(math.Pi*x/2.) + math.Exp(-x/e)
			}
			function := func(x float64) float64 {
				return -math.Cos(math.Pi*x/2.)*(math.Pi*math.Pi*e/4.) -
					math.Pi/2.*math.Sin(math.Pi*x/2.) -
					math.Exp((d-x)/e) - math.Cos(math.Pi*(x-d)/2.)
			}
			phi := func(x float64) float64 {
				return math.Exp(-x / e)
			}

			// Выбор сетки
			var h []float64
			switch job.Request.MeshType {
			case "shishkin":
				h = gridDesign.ShishkinMesh(e, n)
			case "bakhvalov":
				h = gridDesign.BakhvalovaMesh(e, n)
			default:
				h = gridDesign.UniformMesh(n)
			}

			uzel := gridDesign.FindPoints(h, n)

			// Вычисляем phi и phiDelta для модифицированной схемы
			phiVals := make([]float64, n+1)
			phiDeltaVals := make([]float64, n+1)
			for k := 0; k <= n; k++ {
				phiVals[k] = phi(uzel[k])
				phiDeltaVals[k] = phi(uzel[k] - d)
			}

			abcf1 := differenceScheme.ClassicTeylorFormulasScheme(n, e, h, d, function, uzel)
			abcf2 := differenceScheme.ModifiedTeylorFormulasScheme(n, e, h, d, function, uzel, phiVals, phiDeltaVals)

			u1 := thomasMethod.Progonka(abcf1.A, abcf1.B, abcf1.C, abcf1.F, n, e)
			u2 := thomasMethod.Progonka(abcf2.A, abcf2.B, abcf2.C, abcf2.F, n, e)

			a := examineSolution.ErrorNorm(u1, n, solution, uzel)
			b := examineSolution.ErrorNorm(u2, n, solution, uzel)

			classic[i][j] = strings.Replace(fmt.Sprintf("%6.2e", a), ",", ".", -1)
			modified[i][j] = strings.Replace(fmt.Sprintf("%6.2e", b), ",", ".", -1)

			j++
		}
		i++
	}

	// Генерируем PDF
	//timestamp := time.Now().Format("20060102-150405")
	pdfPath := filepath.Join(job.ResultsDir, fmt.Sprintf("%s.tex", job.ID))

	title := fmt.Sprintf("Сетка %s $\\delta = %.2f$",
		meshTypeName(job.Request.MeshType), job.Request.Delta)

	err := latex.Latex(pdfPath, title, classic, modified)

	job.mu.Lock()
	defer job.mu.Unlock()

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		return
	}

	job.Status = "completed"
	job.PDFPath = pdfPath
	job.Classic = classic
	job.Modified = modified
}

func meshTypeName(meshType string) string {
	switch meshType {
	case "shishkin":
		return "Шишкина"
	case "bakhvalov":
		return "Бахвалова"
	default:
		return "Равномерная"
	}
}
