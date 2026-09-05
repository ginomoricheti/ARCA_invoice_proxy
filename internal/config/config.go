package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	ARCA     ARCAConfig
	Auth     AuthConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type ARCAConfig struct {
	Environment     string
	WSAAURL         string
	WSFEURL         string
	CertificatePath string
	KeyPath         string
}

type AuthConfig struct {
	APIKeyPepper   string
	IdempotencyTTL time.Duration
}

type LoggingConfig struct {
	Level  string
	Format string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/arca_proxy?sslmode=disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		ARCA: ARCAConfig{
			Environment:     getEnv("ARCA_ENVIRONMENT", "homologation"),
			WSAAURL:         getEnv("ARCA_WSAA_URL", "https://wsaa.homo.afip.gov.ar/ws/services/LoginCms"),
			WSFEURL:         getEnv("ARCA_WSFE_URL", "https://servicios1.afip.gov.ar/wsfev1/service.asmx"),
			CertificatePath: getEnv("ARCA_CERT_PATH", ""),
			KeyPath:         getEnv("ARCA_KEY_PATH", ""),
		},
		Auth: AuthConfig{
			APIKeyPepper:   getEnv("API_KEY_PEPPER", "change-me-in-production"),
			IdempotencyTTL: getDurationEnv("IDEMPOTENCY_TTL", 24*time.Hour),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
