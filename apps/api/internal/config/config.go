package config

import (
	"errors"
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

func (c *Config) ValidateProduction() error {
	if strings.EqualFold(c.Environment, "production") {
		// 1. Mandatory strong JWT Secret
		if c.JWTSecret == "" ||
			c.JWTSecret == "hostvra-dev-insecure-jwt-secret-key-change-in-production-min64char" ||
			len(c.JWTSecret) < 32 {
			return errors.New("CRITICAL STARTUP ERROR: JWT_SECRET must be explicitly provided in production and be at least 32 characters long")
		}

		// 2. Mandatory non-default Database URL
		if c.DatabaseURL == "" ||
			strings.Contains(c.DatabaseURL, "hostvra_dev_password") {
			return errors.New("CRITICAL STARTUP ERROR: DATABASE_URL must be explicitly configured in production with valid credentials")
		}
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

