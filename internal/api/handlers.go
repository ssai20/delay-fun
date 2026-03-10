package api

import (
	"encoding/json"
	"fun-delay/internal/calculator"
	"fun-delay/internal/models"
	observability "fun-delay/internal/observablity"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"

	"net/http"
)

var (
	logger  *zap.Logger
	metrics *observability.Metrics
)

type CalculationResponse struct {
	JobID    string     `json:"job_id"`
	Status   string     `json:"status"`
	PDFURL   string     `json:"pdf_url,omitempty"`
	Error    string     `json:"error,omitempty"`
	Classic  [][]string `json:"classic,omitempty"`
	Modified [][]string `json:"modified,omitempty"`
}

// Хранилище для отслеживания заданий
var jobs = make(map[string]*calculator.Job)

func calculateHandler(resultsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := observability.StartSpan(r.Context(), "calculateHandler")
		defer span.End()

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.CalculationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			observability.RecordError(span, err)
			logger.Error("Failed to decode request", zap.Error(err))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Добавляем атрибуты к спану
		span.SetAttributes(
			attribute.Float64("request.epsilon_start", req.EpsilonStart),
			attribute.Float64("request.epsilon_min", req.EpsilonMin),
			attribute.Int("request.n_start", req.NStart),
			attribute.Int("request.n_max", req.NMax),
			attribute.Float64("request.delta", req.Delta),
			attribute.String("request.mesh_type", req.MeshType),
		)

		// Обновляем метрики
		metrics.CalculationsTotal.Inc()
		metrics.EpsilonValues.Observe(req.EpsilonMin)
		metrics.MeshSizes.Observe(float64(req.NMax))

		// Валидация
		if req.EpsilonStart <= 0 {
			req.EpsilonStart = 1.0
		}
		if req.EpsilonMin <= 0 {
			req.EpsilonMin = 1e-8
		}
		if req.NStart < 128 {
			req.NStart = 128
		}
		if req.NMax < req.NStart {
			req.NMax = req.NStart * 2
		}
		if req.MeshType == "" {
			req.MeshType = "uniform"
		}

		// Создаем задание
		job := calculator.NewJob(req, resultsDir, logger, metrics)
		jobs[job.ID] = job

		metrics.JobsActive.Inc()
		metrics.JobsTotal.WithLabelValues("pending", req.MeshType).Inc()

		// Запускаем вычисления в фоне
		go calculator.RunJob(job, ctx)

		// Возвращаем ID задания
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(CalculationResponse{
			JobID:  job.ID,
			Status: "processing",
		})
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	_, span := observability.StartSpan(r.Context(), "statusHandler")
	defer span.End()

	jobID := strings.TrimPrefix(r.URL.Path, "/api/status/")
	span.SetAttributes(attribute.String("job.id", jobID))

	job, exists := jobs[jobID]

	if !exists {
		observability.RecordError(span, http.ErrNoLocation)
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	job.RLock()
	defer job.RUnlock()

	resp := CalculationResponse{
		JobID:  job.ID,
		Status: job.Status,
	}

	if job.Status == "completed" && job.PDFPath != "" {
		resp.PDFURL = "/results/" + jobID + ".pdf"
		resp.Classic = job.Classic
		resp.Modified = job.Modified
	} else if job.Status == "failed" {
		resp.Error = job.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	logger.Debug("Status check",
		zap.String("job_id", jobID),
		zap.String("status", job.Status),
	)
}

func meshTypesHandler(w http.ResponseWriter, r *http.Request) {
	_, span := observability.StartSpan(r.Context(), "meshTypesHandler")
	defer span.End()

	types := []map[string]string{
		{"id": "uniform", "name": "Равномерная сетка", "description": "Обычная равномерная сетка"},
		{"id": "shishkin", "name": "Сетка Шишкина", "description": "Адаптивная сетка для пограничного слоя"},
		{"id": "bakhvalov", "name": "Сетка Бахвалова", "description": "Специальная сетка Бахвалова"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy"}`))
}

func SetupRoutes(resultsDir string, log *zap.Logger, m *observability.Metrics) *http.ServeMux {
	logger = log
	metrics = m

	router := http.NewServeMux()

	// Добавляем middleware для метрик
	// 👇 СПЕЦИАЛЬНЫЙ ОБРАБОТЧИК для статических файлов с правильными MIME-типами
	router.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		_, span := observability.StartSpan(r.Context(), "staticHandler")
		defer span.End()

		// Убираем префикс /static/
		filePath := "." + r.URL.Path

		// Определяем MIME-тип по расширению
		if strings.HasSuffix(filePath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(filePath, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(filePath, ".html") {
			w.Header().Set("Content-Type", "text/html")
		}

		http.ServeFile(w, r, filePath)
	})

	router.HandleFunc("/", homeHandler)

	// Статические файлы (PDF результаты)
	router.Handle("/results/", http.StripPrefix("/results/",
		http.FileServer(http.Dir(resultsDir))))

	// Добавить прямой download URL
	router.HandleFunc("/download/", downloadHandler(resultsDir))

	// API endpoints
	router.HandleFunc("/api/calculate", calculateHandler(resultsDir))
	router.HandleFunc("/api/status/", statusHandler)
	router.HandleFunc("/api/mesh-types", meshTypesHandler)
	router.HandleFunc("/health", healthHandler)

	// Возвращаем router, но используем middleware
	return router
}

func downloadHandler(resultsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := observability.StartSpan(r.Context(), "downloadHandler")
		defer span.End()

		filename := strings.TrimPrefix(r.URL.Path, "/download/")
		if !strings.HasSuffix(filename, ".pdf") {
			filename += ".pdf"
		}

		filepath := filepath.Join(resultsDir, filename)

		// Убеждаемся, что файл существует
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			observability.RecordError(span, err)
			logger.Error("File not found", zap.String("file", filepath))

			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Отдаем файл
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, filepath)

		logger.Info("File downloaded", zap.String("file", filename))
	}
}

// Обновите homeHandler для правильного MIME-типа HTML
func homeHandler(w http.ResponseWriter, r *http.Request) {
	_, span := observability.StartSpan(r.Context(), "homeHandler")
	defer span.End()

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, "./static/index.html")
}
