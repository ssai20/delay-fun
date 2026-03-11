package observability

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" // Регистрирует pprof handlers
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"go.uber.org/zap"
)

func InitProfiler(cfg *Config) {
	if !cfg.ProfilerEnabled {
		return
	}

	// Настройка профилирования CPU для периодической записи
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := takeCPUProfile(); err != nil {
				GetLogger().Error("Failed to take CPU profile", zap.Error(err))
			}
		}
	}()

	// Запуск HTTP сервера для pprof
	go func() {
		mux := http.NewServeMux()

		// Регистрируем стандартные pprof handlers
		mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})

		// Добавляем кастомные эндпоинты
		mux.HandleFunc(cfg.ProfilerPath+"heap", func(w http.ResponseWriter, r *http.Request) {
			pprof.Lookup("heap").WriteTo(w, 1)
		})

		mux.HandleFunc(cfg.ProfilerPath+"goroutine", func(w http.ResponseWriter, r *http.Request) {
			pprof.Lookup("goroutine").WriteTo(w, 1)
		})

		mux.HandleFunc(cfg.ProfilerPath+"allocs", func(w http.ResponseWriter, r *http.Request) {
			pprof.Lookup("allocs").WriteTo(w, 1)
		})

		mux.HandleFunc(cfg.ProfilerPath+"threadcreate", func(w http.ResponseWriter, r *http.Request) {
			pprof.Lookup("threadcreate").WriteTo(w, 1)
		})

		mux.HandleFunc(cfg.ProfilerPath+"block", func(w http.ResponseWriter, r *http.Request) {
			pprof.Lookup("block").WriteTo(w, 1)
		})

		mux.HandleFunc(cfg.ProfilerPath+"mutex", func(w http.ResponseWriter, r *http.Request) {
			pprof.Lookup("mutex").WriteTo(w, 1)
		})

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.ProfilerPort),
			Handler: mux,
		}

		GetLogger().Info("Profiler server started",
			zap.Int("port", cfg.ProfilerPort),
			zap.String("path", cfg.ProfilerPath),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			GetLogger().Error("Profiler server error", zap.Error(err))
		}
	}()

	// Включаем сбор статистики блокировок
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
}

func takeCPUProfile() error {
	// Сохраняйте в /tmp вместо текущей директории
	f, err := os.Create(fmt.Sprintf("/tmp/cpu_profile_%s.pprof",
		time.Now().Format("20060102_150405")))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()

	// Собираем профиль в течение 30 секунд
	time.Sleep(30 * time.Second)

	return nil
}
