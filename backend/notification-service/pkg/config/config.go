package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default database pool sizing, per replica. Each service has its own
// Postgres (max_connections defaults to 100, a few of which are reserved for
// superuser/operator/backup use), and the HPA can run up to 4 replicas, so a
// replica gets 20: 4 x 20 = 80 leaves headroom. Left unset, database/sql
// opens connections without limit, and 4 busy replicas can exceed
// max_connections ("too many clients already").
const (
	defaultDBMaxOpenConns    = 20
	defaultDBMaxIdleConns    = 5
	defaultDBConnMaxLifetime = 30 * time.Minute
)

type Config struct {
	AppPort          string
	RabbitMQURL      string
	RabbitQueueType  string
	JWTSecret        string
	AllowedOrigins   []string
	WSRateLimitRPS   float64
	WSRateLimitBurst float64
	TrustedProxies   []string
	RedisURL         string
	MaxWSConnections int
	DBHost           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBPort           string

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

var AppConfig Config

func Load() {
	AppConfig = Config{
		AppPort:          os.Getenv("APP_PORT"),
		RabbitMQURL:      os.Getenv("RABBITMQ_URL"),
		RabbitQueueType:  queueTypeOrDefault(os.Getenv("RABBITMQ_QUEUE_TYPE")),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AllowedOrigins:   parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
		WSRateLimitRPS:   parseFloatOrDefault(os.Getenv("WS_RATE_LIMIT_RPS"), 5),
		WSRateLimitBurst: parseFloatOrDefault(os.Getenv("WS_RATE_LIMIT_BURST"), 10),
		TrustedProxies:   splitCSV(os.Getenv("TRUSTED_PROXIES")),
		RedisURL:         os.Getenv("REDIS_URL"),
		MaxWSConnections: parseIntOrDefault(os.Getenv("MAX_WS_CONNECTIONS"), 1000),
		DBHost:           os.Getenv("DB_HOST"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		DBPort:           os.Getenv("DB_PORT"),

		DBMaxOpenConns:    parseIntOrDefault(os.Getenv("DB_MAX_OPEN_CONNS"), defaultDBMaxOpenConns),
		DBMaxIdleConns:    parseIntOrDefault(os.Getenv("DB_MAX_IDLE_CONNS"), defaultDBMaxIdleConns),
		DBConnMaxLifetime: parseDurationOrDefault(os.Getenv("DB_CONN_MAX_LIFETIME"), defaultDBConnMaxLifetime),
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
}

// parseOrigins splits a comma-separated ALLOWED_ORIGINS value. Falls back to
// "*" (with a warning) so local/dev setups keep working without extra config.
func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		log.Println("WARNING: ALLOWED_ORIGINS not set, defaulting to '*' (not safe for production)")
		return []string{"*"}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			origins = append(origins, p)
		}
	}
	return origins
}

// queueTypeOrDefault defaults to classic, which is what existing queues (and
// docker compose) use; quorum is opted into with RABBITMQ_QUEUE_TYPE=quorum.
// Validation happens in main so config stays free of the consumer package.
func queueTypeOrDefault(raw string) string {
	if raw == "" {
		return "classic"
	}
	return raw
}

// splitCSV splits a comma-separated env value, dropping blanks.
func splitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
