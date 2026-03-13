package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config contains all runtime settings loaded from environment variables.
type Config struct {
	Env              string
	HTTPPort         int
	DatabaseURL      string
	JWTSecret        string
	AppBaseURL       string
	MailFrom         string
	AdminEmails      []string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	EmailVerifyTTL   time.Duration
	PasswordResetTTL time.Duration
	LogLevel         string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
}

type rawConfig struct {
	Env              string `validate:"required,oneof=development test production"`
	HTTPPort         int    `validate:"required,min=1,max=65535"`
	DatabaseURL      string `validate:"required"`
	JWTSecret        string `validate:"required,min=32"`
	AppBaseURL       string `validate:"required,url"`
	MailFrom         string `validate:"required,email,max=254"`
	AccessTokenTTL   string `validate:"required"`
	RefreshTokenTTL  string `validate:"required"`
	EmailVerifyTTL   string `validate:"required"`
	PasswordResetTTL string `validate:"required"`
	AdminEmails      string
	LogLevel         string `validate:"required,oneof=debug info warn error"`
	ReadTimeout      string `validate:"required"`
	WriteTimeout     string `validate:"required"`
}

// Load parses environment variables and validates configuration.
func Load() (Config, error) {
	raw := rawConfig{
		Env:              env("APP_ENV", "development"),
		HTTPPort:         envInt("HTTP_PORT", 8080),
		DatabaseURL:      env("DATABASE_URL", ""),
		JWTSecret:        env("JWT_SECRET", ""),
		AppBaseURL:       env("APP_BASE_URL", "http://localhost:5173"),
		MailFrom:         env("MAIL_FROM", "no-reply@marketplace.local"),
		AdminEmails:      env("ADMIN_EMAILS", ""),
		AccessTokenTTL:   env("ACCESS_TOKEN_TTL", "15m"),
		RefreshTokenTTL:  env("REFRESH_TOKEN_TTL", "720h"),
		EmailVerifyTTL:   env("EMAIL_VERIFY_TTL", "24h"),
		PasswordResetTTL: env("PASSWORD_RESET_TTL", "1h"),
		LogLevel:         strings.ToLower(env("LOG_LEVEL", "info")),
		ReadTimeout:      env("HTTP_READ_TIMEOUT", "10s"),
		WriteTimeout:     env("HTTP_WRITE_TIMEOUT", "15s"),
	}

	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(raw); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	accessTTL, err := time.ParseDuration(raw.AccessTokenTTL)
	if err != nil {
		return Config{}, fmt.Errorf("parse ACCESS_TOKEN_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(raw.RefreshTokenTTL)
	if err != nil {
		return Config{}, fmt.Errorf("parse REFRESH_TOKEN_TTL: %w", err)
	}
	emailVerifyTTL, err := time.ParseDuration(raw.EmailVerifyTTL)
	if err != nil {
		return Config{}, fmt.Errorf("parse EMAIL_VERIFY_TTL: %w", err)
	}
	passwordResetTTL, err := time.ParseDuration(raw.PasswordResetTTL)
	if err != nil {
		return Config{}, fmt.Errorf("parse PASSWORD_RESET_TTL: %w", err)
	}
	if refreshTTL <= accessTTL {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL")
	}
	if emailVerifyTTL <= 0 {
		return Config{}, fmt.Errorf("EMAIL_VERIFY_TTL must be greater than zero")
	}
	if passwordResetTTL <= 0 {
		return Config{}, fmt.Errorf("PASSWORD_RESET_TTL must be greater than zero")
	}

	readTimeout, err := time.ParseDuration(raw.ReadTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := time.ParseDuration(raw.WriteTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_WRITE_TIMEOUT: %w", err)
	}

	cfg := Config{
		Env:              raw.Env,
		HTTPPort:         raw.HTTPPort,
		DatabaseURL:      raw.DatabaseURL,
		JWTSecret:        raw.JWTSecret,
		AppBaseURL:       raw.AppBaseURL,
		MailFrom:         raw.MailFrom,
		AdminEmails:      parseEmailList(raw.AdminEmails),
		AccessTokenTTL:   accessTTL,
		RefreshTokenTTL:  refreshTTL,
		EmailVerifyTTL:   emailVerifyTTL,
		PasswordResetTTL: passwordResetTTL,
		LogLevel:         raw.LogLevel,
		ReadTimeout:      readTimeout,
		WriteTimeout:     writeTimeout,
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseEmailList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		email := strings.ToLower(strings.TrimSpace(part))
		if email == "" {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}
		result = append(result, email)
	}
	return result
}
