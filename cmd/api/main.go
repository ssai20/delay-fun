package main

import (
	"context"
	"fun-delay/internal/api"
	observability "fun-delay/internal/observablity"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	// Оптимизация под доступные CPU
	_, _ = maxprocs.Set()

	// Загрузка конфигурации
	cfg := observability.LoadConfig()

	// Инициализация логирования
	logger := observability.InitLogger(cfg)
	defer logger.Sync()

	// Инициализация трассировки
	tp, err := observability.InitTracing(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize tracing", zap.Error(err))
	}
	if tp != nil {
		defer tp.Shutdown(context.Background())
	}

	// Инициализация метрик
	metrics := observability.InitMetrics(cfg)

	// Инициализация профилировщика
	observability.InitProfiler(cfg)

	// Запуск сервера метрик
	metricsServer := observability.StartMetricsServer(cfg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	// Создаем директорию для результатов
	resultsDir := os.Getenv("RESULTS_DIR")
	if resultsDir == "" {
		resultsDir = "./results"
	}
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		log.Fatal("Failed to create results directory:", err)
	}

	// Настраиваем маршруты
	router := api.SetupRoutes(resultsDir, logger, metrics)

	// Создаем HTTP сервер с middleware
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Server starting", zap.String("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	<-done
	logger.Info("Server stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server shutdown failed", zap.Error(err))
	}

	if metricsServer != nil {
		if err := metricsServer.Shutdown(ctx); err != nil {
			logger.Error("Metrics server shutdown failed", zap.Error(err))
		}
	}

	logger.Info("Server stopped")
}
