package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment          string
	Port                 string
	Host                 string
	AppURL               string
	APIURL               string
	DatabaseURL          string
	RedisURL             string
	JWTSecret            string
	JWTElementsHours     time.Duration
	RefreshTokenDays     time.Duration
	CORSAllowedOrigins   []string
	SystemEdition        string
	LogLevel             string
	LogFormat            string
}

func Load() *Config {
	return &Config{
		Environment:        getEnv("ENVIRONMENT", "development"),
		Port:               getEnv("PORT", "8080"),
		Host:               getEnv("HOST", "0.0.0.0"),
		AppURL:             getEnv("APP_URL", "http://localhost:3000"),
		APIURL:             getEnv("API_URL", "http://localhost:8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://hostvra:hostvra_dev_password@localhost:5432/hostvra?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:          getEnv("JWT_SECRET", "hostvra-dev-insecure-jwt-secret-key-change-in-production-min64char"),
		JWTElementsHours:   time.Duration(getEnvInt("JWT_EXPIRATION_HOURS", 24)) * time.Hour,
		RefreshTokenDays:   time.Duration(getEnvInt("REFRESH_TOKEN_EXPIRATION_DAYS", 7)) * 24 * time.Hour,
		CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"), ","),
		SystemEdition:      getEnv("HOSTVRA_SYSTEM_EDITION", "community"),
		LogLevel:           getEnv("LOG_LEVEL", "debug"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return fallback
}
