package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName string
	HTTPPort    int
	GRPCPort    int
	DatabaseURL string
	RedisURL    string
	KafkaBrokers []string
}

func Load(serviceName string) Config {
	return Config{
		ServiceName:  serviceName,
		HTTPPort:     envInt("HTTP_PORT", 8080),
		GRPCPort:     envInt("GRPC_PORT", 9090),
		DatabaseURL:  env("DATABASE_URL", "postgres://eventflow:eventflow@localhost:5432/eventflow?sslmode=disable"),
		RedisURL:     env("REDIS_URL", "redis://localhost:6379/0"),
		KafkaBrokers: envSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		return []string{v}
	}
	return fallback
}

func DefaultRetryPolicy() RetryPolicy {
	if os.Getenv("DEMO_MODE") == "true" {
		return RetryPolicy{
			MaxAttempts:    envInt("RETRY_MAX_ATTEMPTS", 3),
			InitialBackoff: time.Duration(envInt("RETRY_INITIAL_MS", 2000)) * time.Millisecond,
			MaxBackoff:     time.Duration(envInt("RETRY_MAX_MS", 8000)) * time.Millisecond,
			Multiplier:     2.0,
			Timeout:        30 * time.Second,
		}
	}
	return RetryPolicy{
		MaxAttempts:    envInt("RETRY_MAX_ATTEMPTS", 5),
		InitialBackoff: time.Duration(envInt("RETRY_INITIAL_MS", 1000)) * time.Millisecond,
		MaxBackoff:     5 * time.Minute,
		Multiplier:     2.0,
		Timeout:        30 * time.Second,
	}
}

type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
	Timeout        time.Duration
}
