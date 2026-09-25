package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Default database pool sizing, per replica. Each service has its own
// Postgres (max_connections defaults to 100, a few of which are reserved for
// superuser/operator/backup use), and the HPA can run up to 4 replicas, so a
// replica gets 20: 4 x 20 = 80 leaves headroom. This service used to allow
// 100 per replica, which alone is 400 across 4 replicas.
const (
	defaultDBMaxOpenConns    = 20
	defaultDBMaxIdleConns    = 5
	defaultDBConnMaxLifetime = 30 * time.Minute
)

type Config struct {
	AppPort        string
	JWTSecret      string
	DBHost         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBPort         string
	InternalSecret string
	EnableSwagger  bool
	AuthRateLimitRPS   float64
	AuthRateLimitBurst float64
	RedisURL           string

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:            os.Getenv("APP_PORT"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		DBHost:             os.Getenv("DB_HOST"),
		DBUser:             os.Getenv("DB_USER"),
		DBPassword:         os.Getenv("DB_PASSWORD"),
		DBName:             os.Getenv("DB_NAME"),
		DBPort:             os.Getenv("DB_PORT"),
		InternalSecret:     os.Getenv("INTERNAL_SHARED_SECRET"),
		EnableSwagger:      os.Getenv("ENABLE_SWAGGER") == "true",
		AuthRateLimitRPS:   parseFloatOrDefault(os.Getenv("AUTH_RATE_LIMIT_RPS"), 5),
		AuthRateLimitBurst: parseFloatOrDefault(os.Getenv("AUTH_RATE_LIMIT_BURST"), 10),
		RedisURL:           os.Getenv("REDIS_URL"),

		DBMaxOpenConns:    parseIntOrDefault(os.Getenv("DB_MAX_OPEN_CONNS"), defaultDBMaxOpenConns),
		DBMaxIdleConns:    parseIntOrDefault(os.Getenv("DB_MAX_IDLE_CONNS"), defaultDBMaxIdleConns),
		DBConnMaxLifetime: parseDurationOrDefault(os.Getenv("DB_CONN_MAX_LIFETIME"), defaultDBConnMaxLifetime),

		BootstrapAdminEmail:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}

	if AppConfig.AppPort == "" {
		log.Fatal("APP_PORT is required")
	}

	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if AppConfig.DBHost == "" {
		log.Fatal("DB_HOST is required")
	}

	if AppConfig.InternalSecret == "" {
		log.Fatal("INTERNAL_SHARED_SECRET is required")
	}
}

func parseIntOrDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func parseDurationOrDefault(raw string, def time.Duration) time.Duration {
	if raw == "" {
		return def
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func parseFloatOrDefault(raw string, def float64) float64 {
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return def
	}
	return v
}
