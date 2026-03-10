package calculator

import (
	"context"
	"fmt"
	"fun-delay/internal/differenceScheme"
	"fun-delay/internal/examineSolution"
	"fun-delay/internal/gridDesign"
	"fun-delay/internal/latex"
	"fun-delay/internal/models"
	observability "fun-delay/internal/observability"
	"fun-delay/internal/thomasMethod"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Job struct {
	ID       string
	Request  models.CalculationRequest
	Status   string
	PDFPath  string
	Error    string
	Classic  [][]string
	Modified [][]string
	sync.RWMutex
	ResultsDir string
	logger     *zap.Logger
	metrics    *observability.Metrics
	createdAt  time.Time
}

func NewJob(req models.CalculationRequest, resultsDir string, logger *zap.Logger, metrics *observability.Metrics) *Job {
	return &Job{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		Request:    req,
		Status:     "pending",
		ResultsDir: resultsDir,
		logger:     logger,
		metrics:    metrics,
		createdAt:  time.Now(),
	}
}

func RunJob(job *Job, ctx context.Context) {
	ctx, span := observability.StartSpan(ctx, "RunJob")
	defer span.End()

	span.SetAttributes(
		attribute.String("job.id", job.ID),
		attribute.String("job.mesh_type", job.Request.MeshType),
		attribute.Float64("job.epsilon_min", job.Request.EpsilonMin),
		attribute.Int("job.n_max", job.Request.NMax),
	)

	job.Lock()
	job.Status = "processing"
	job.Unlock()

	job.metrics.JobsTotal.WithLabelValues("processing", job.Request.MeshType).Inc()
	startTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			observability.RecordError(span, fmt.Errorf("%v", r))

			job.Lock()
			job.Status = "failed"
			job.Error = fmt.Sprintf("Panic: %v", r)
			job.Unlock()

			job.metrics.JobsTotal.WithLabelValues("failed", job.Request.MeshType).Inc()
			job.logger.Error("Job failed with panic",
				zap.String("job_id", job.ID),
				zap.Any("panic", r),
			)
		}
	}()

	job.logger.Info("Starting job",
		zap.String("job_id", job.ID),
		zap.Any("request", job.Request),
	)

	classic := make([][]string, 9)
	modified := make([][]string, 9)

	i := 0
	for e := job.Request.EpsilonStart; e >= job.Request.EpsilonMin; e /= 10. {
		classic[i] = make([]string, 5)
		modified[i] = make([]string, 5)
		j := 0

		for n := job.Request.NStart; n <= job.Request.NMax; n *= 2 {
			// Создаем span для каждой итерации
			_, iterSpan := observability.StartSpan(ctx, fmt.Sprintf("iteration_eps_%e_n_%d", e, n))

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

			iterSpan.SetAttributes(
				attribute.Float64("iteration.error_classic", a),
				attribute.Float64("iteration.error_modified", b),
			)
			iterSpan.End()

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

	duration := time.Since(startTime).Seconds()
	job.metrics.JobsDuration.WithLabelValues(job.Request.MeshType).Observe(duration)
	job.metrics.JobsActive.Dec()

	job.Lock()
	defer job.Unlock()

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()

		job.metrics.JobsTotal.WithLabelValues("failed", job.Request.MeshType).Inc()

		observability.RecordError(span, err)
		job.logger.Error("Job failed",
			zap.String("job_id", job.ID),
			zap.Error(err),
		)
		return
	}

	job.Status = "completed"
	job.PDFPath = pdfPath
	job.Classic = classic
	job.Modified = modified

	job.metrics.JobsTotal.WithLabelValues("completed", job.Request.MeshType).Inc()

	job.logger.Info("Job completed",
		zap.String("job_id", job.ID),
		zap.Float64("duration_seconds", duration),
	)

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
