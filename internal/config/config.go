// Package config — конфигурация Dashboard service.
package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string // адрес WS/HTTP сервера
	AnalyticsAddr   string // gRPC-адрес Analytics service
	WebDist         string // путь к собранной SPA (web/dist); "" = статику не отдаём
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8082"),
		AnalyticsAddr:   getEnv("ANALYTICS_ADDR", "localhost:50051"),
		WebDist:         os.Getenv("WEB_DIST"),
		ShutdownTimeout: 10 * time.Second,
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
