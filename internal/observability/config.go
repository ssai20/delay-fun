package observability

import (
	"os"
	"strconv"
)

type Config struct {
	// Логирование
	LogLevel      string
	LogFormat     string // json или console
	LogOutputPath string

	// Трассировка
	TraceEnabled     bool
	TraceEndpoint    string
	TraceServiceName string
	TraceEnvironment string

	// Метрики
	MetricsEnabled bool
	MetricsPath    string
	MetricsPort    int

	// Профилировщик
	ProfilerEnabled bool
	ProfilerPort    int
	ProfilerPath    string
}

func LoadConfig() *Config {
	cfg := &Config{
		// Логирование
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LogFormat:     getEnv("LOG_FORMAT", "json"),
		LogOutputPath: getEnv("LOG_OUTPUT", "stdout"),

		// Трассировка
		TraceEnabled:     getEnvBool("TRACE_ENABLED", false),
		TraceEndpoint:    getEnv("TRACE_ENDPOINT", "http://jaeger:4318"),
		TraceServiceName: getEnv("TRACE_SERVICE_NAME", "fun-delay"),
		TraceEnvironment: getEnv("TRACE_ENVIRONMENT", "production"),

		// Метрики
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
		MetricsPath:    getEnv("METRICS_PATH", "/metrics"),
		MetricsPort:    getEnvInt("METRICS_PORT", 9090),

		// Профилировщик
		ProfilerEnabled: getEnvBool("PROFILER_ENABLED", false),
		ProfilerPort:    getEnvInt("PROFILER_PORT", 6060),
		ProfilerPath:    getEnv("PROFILER_PATH", "/debug/pprof/"),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
