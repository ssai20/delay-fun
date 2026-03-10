package observability

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	// HTTP метрики
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	// Бизнес метрики
	JobsTotal         *prometheus.CounterVec
	JobsDuration      *prometheus.HistogramVec
	JobsActive        prometheus.Gauge
	CalculationsTotal prometheus.Counter

	// Системные метрики
	EpsilonValues   prometheus.Histogram
	MeshSizes       prometheus.Histogram
	memoryUsage     prometheus.GaugeFunc
	goroutinesCount prometheus.GaugeFunc
}

var metrics *Metrics

func InitMetrics(cfg *Config) *Metrics {
	metrics = &Metrics{
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		httpRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Current number of HTTP requests in flight",
			},
		),
		JobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_total",
				Help: "Total number of jobs",
			},
			[]string{"status", "mesh_type"},
		),
		JobsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "jobs_duration_seconds",
				Help:    "Job duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"mesh_type"},
		),
		JobsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "jobs_active",
				Help: "Current number of active jobs",
			},
		),
		CalculationsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "calculations_total",
				Help: "Total number of calculations performed",
			},
		),
		EpsilonValues: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "epsilon_values",
				Help:    "Distribution of epsilon values used in calculations",
				Buckets: prometheus.ExponentialBuckets(1e-8, 10, 10),
			},
		),
		MeshSizes: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "mesh_sizes",
				Help:    "Distribution of mesh sizes (N) used in calculations",
				Buckets: []float64{128, 256, 512, 1024, 2048, 4096},
			},
		),
	}

	// Добавляем системные метрики через GaugeFunc
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.Alloc)
		},
	))

	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "goroutines_count",
			Help: "Number of running goroutines",
		},
		func() float64 {
			return float64(runtime.NumGoroutine())
		},
	))

	return metrics
}

func GetMetrics() *Metrics {
	if metrics == nil {
		return &Metrics{} // Возвращаем пустые метрики если не инициализированы
	}
	return metrics
}

// Middleware для сбора HTTP метрик
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		m.httpRequestsInFlight.Inc()
		defer m.httpRequestsInFlight.Dec()

		// Создаем кастомный ResponseWriter для захвата статуса
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start).Seconds()
		m.httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			fmt.Sprintf("%d", wrapper.statusCode),
		).Inc()

		m.httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Функция для запуска HTTP сервера с метриками
func StartMetricsServer(cfg *Config) *http.Server {
	if !cfg.MetricsEnabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.MetricsPath, promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			GetLogger().Error("Metrics server error", zap.Error(err))
		}
	}()

	GetLogger().Info("Metrics server started",
		zap.Int("port", cfg.MetricsPort),
		zap.String("path", cfg.MetricsPath),
	)

	return server
}
