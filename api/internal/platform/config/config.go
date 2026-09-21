// Package config loads the runtime configuration from the environment.
//
// Every value has a development friendly default so a fresh checkout runs with
// zero configuration; production values are validated at startup.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Recognized application environments.
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTest        = "test"
)

// developmentSecret is the placeholder used when JWT_SECRET is not configured.
// It is rejected in production.
const developmentSecret = "dev-insecure-secret-change-me"

// Config is the root configuration of the API.
type Config struct {
	Env             string
	HTTPAddr        string
	LogLevel        string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CORSOrigins     []string
	Storage         Storage
}

// Storage describes the S3 compatible object storage used for post media.
type Storage struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	UseSSL        bool
	PublicBaseURL string
}

// IsProduction reports whether the API runs with production settings.
func (c Config) IsProduction() bool { return c.Env == EnvProduction }

// Load reads the configuration from the environment.
func Load() (Config, error) {
	accessTTL, err := durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := durationEnv("REFRESH_TOKEN_TTL", 720*time.Hour)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Env:             strings.ToLower(env("APP_ENV", EnvDevelopment)),
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		LogLevel:        env("LOG_LEVEL", "info"),
		DatabaseURL:     env("DATABASE_URL", "postgres://saf:saf@localhost:5432/saf?sslmode=disable"),
		RedisURL:        env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:       env("JWT_SECRET", developmentSecret),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		CORSOrigins:     listEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		Storage: Storage{
			Endpoint:      env("S3_ENDPOINT", "localhost:9000"),
			Region:        env("S3_REGION", "us-east-1"),
			Bucket:        env("S3_BUCKET", "saf-media"),
			AccessKey:     env("S3_ACCESS_KEY", "safminio"),
			SecretKey:     env("S3_SECRET_KEY", "safminio123"),
			UseSSL:        boolEnv("S3_USE_SSL", false),
			PublicBaseURL: env("S3_PUBLIC_BASE_URL", "http://localhost:9000/saf-media"),
		},
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	switch c.Env {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		return fmt.Errorf("APP_ENV must be one of %q, %q or %q", EnvDevelopment, EnvProduction, EnvTest)
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL must be set")
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be positive")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be positive")
	}
	if c.IsProduction() {
		if c.JWTSecret == developmentSecret || len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be a random string of at least 32 characters in production")
		}
	}
	return nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func boolEnv(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration (for example 15m or 720h): %w", key, err)
	}
	return value, nil
}

func listEnv(key string, fallback []string) []string {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}
