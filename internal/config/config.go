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
	Env                           string
	HTTPPort                      int
	DatabaseURL                   string
	JWTSecret                     string
	AppBaseURL                    string
	MailFrom                      string
	AdminEmails                   []string
	AccessTokenTTL                time.Duration
	RefreshTokenTTL               time.Duration
	EmailVerifyTTL                time.Duration
	PasswordResetTTL              time.Duration
	LoginFailureWindow            time.Duration
	LoginMaxAttempts              int
	LoginLockoutDuration          time.Duration
	AuthRegisterRateLimit         int
	AuthRegisterRateWindow        time.Duration
	AuthLoginRateLimit            int
	AuthLoginRateWindow           time.Duration
	AuthRefreshRateLimit          int
	AuthRefreshRateWindow         time.Duration
	AuthPasswordResetRateLimit    int
	AuthPasswordResetRateWindow   time.Duration
	AuthVerifyEmailRateLimit      int
	AuthVerifyEmailRateWindow     time.Duration
	AuthCookieMode                bool
	AuthCookieSecure              bool
	AuthCookieDomain              string
	AuthCookieSameSite            string
	AuthCSRFEnabled               bool
	LogLevel                      string
	ReadTimeout                   time.Duration
	WriteTimeout                  time.Duration
	JobsEnabled                   bool
	CleanupInterval               time.Duration
	EmailPollInterval             time.Duration
	EmailLockTTL                  time.Duration
	EmailBatchSize                int
	EmailMaxAttempts              int
	EmailRetention                time.Duration
	StatsRefreshInterval          time.Duration
	RecommendationRefreshInterval time.Duration
	RecommendationActivityWindow  time.Duration
	RecommendationUserBatchSize   int
	RecommendationLimit           int
}

type rawConfig struct {
	Env                           string `validate:"required,oneof=development test production"`
	HTTPPort                      int    `validate:"required,min=1,max=65535"`
	DatabaseURL                   string `validate:"required"`
	JWTSecret                     string `validate:"required,min=32"`
	AppBaseURL                    string `validate:"required,url"`
	MailFrom                      string `validate:"required,email,max=254"`
	AccessTokenTTL                string `validate:"required"`
	RefreshTokenTTL               string `validate:"required"`
	EmailVerifyTTL                string `validate:"required"`
	PasswordResetTTL              string `validate:"required"`
	LoginFailureWindow            string `validate:"required"`
	LoginMaxAttempts              int    `validate:"required,min=1,max=50"`
	LoginLockoutDuration          string `validate:"required"`
	AuthRegisterRateLimit         int    `validate:"required,min=1,max=1000"`
	AuthRegisterRateWindow        string `validate:"required"`
	AuthLoginRateLimit            int    `validate:"required,min=1,max=1000"`
	AuthLoginRateWindow           string `validate:"required"`
	AuthRefreshRateLimit          int    `validate:"required,min=1,max=5000"`
	AuthRefreshRateWindow         string `validate:"required"`
	AuthPasswordResetRateLimit    int    `validate:"required,min=1,max=1000"`
	AuthPasswordResetRateWindow   string `validate:"required"`
	AuthVerifyEmailRateLimit      int    `validate:"required,min=1,max=1000"`
	AuthVerifyEmailRateWindow     string `validate:"required"`
	AuthCookieMode                bool
	AuthCookieSecure              bool
	AuthCookieDomain              string
	AuthCookieSameSite            string `validate:"required,oneof=lax strict none"`
	AuthCSRFEnabled               bool
	AdminEmails                   string
	LogLevel                      string `validate:"required,oneof=debug info warn error"`
	ReadTimeout                   string `validate:"required"`
	WriteTimeout                  string `validate:"required"`
	JobsEnabled                   bool
	CleanupInterval               string `validate:"required"`
	EmailPollInterval             string `validate:"required"`
	EmailLockTTL                  string `validate:"required"`
	EmailBatchSize                int    `validate:"required,min=1,max=500"`
	EmailMaxAttempts              int    `validate:"required,min=1,max=20"`
	EmailRetention                string `validate:"required"`
	StatsRefreshInterval          string `validate:"required"`
	RecommendationRefreshInterval string `validate:"required"`
	RecommendationActivityWindow  string `validate:"required"`
	RecommendationUserBatchSize   int    `validate:"required,min=1,max=5000"`
	RecommendationLimit           int    `validate:"required,min=1,max=100"`
}

// Load parses environment variables and validates configuration.
func Load() (Config, error) {
	raw := rawConfig{
		Env:                           env("APP_ENV", "development"),
		HTTPPort:                      envInt("HTTP_PORT", 8080),
		DatabaseURL:                   env("DATABASE_URL", ""),
		JWTSecret:                     env("JWT_SECRET", ""),
		AppBaseURL:                    env("APP_BASE_URL", "http://localhost:5173"),
		MailFrom:                      env("MAIL_FROM", "no-reply@marketplace.local"),
		AdminEmails:                   env("ADMIN_EMAILS", ""),
		AccessTokenTTL:                env("ACCESS_TOKEN_TTL", "15m"),
		RefreshTokenTTL:               env("REFRESH_TOKEN_TTL", "720h"),
		EmailVerifyTTL:                env("EMAIL_VERIFY_TTL", "24h"),
		PasswordResetTTL:              env("PASSWORD_RESET_TTL", "1h"),
		LoginFailureWindow:            env("AUTH_LOGIN_FAILURE_WINDOW", "15m"),
		LoginMaxAttempts:              envInt("AUTH_LOGIN_MAX_ATTEMPTS", 5),
		LoginLockoutDuration:          env("AUTH_LOGIN_LOCKOUT_DURATION", "15m"),
		AuthRegisterRateLimit:         envInt("AUTH_RATE_LIMIT_REGISTER", 5),
		AuthRegisterRateWindow:        env("AUTH_RATE_LIMIT_REGISTER_WINDOW", "1m"),
		AuthLoginRateLimit:            envInt("AUTH_RATE_LIMIT_LOGIN", 10),
		AuthLoginRateWindow:           env("AUTH_RATE_LIMIT_LOGIN_WINDOW", "1m"),
		AuthRefreshRateLimit:          envInt("AUTH_RATE_LIMIT_REFRESH", 30),
		AuthRefreshRateWindow:         env("AUTH_RATE_LIMIT_REFRESH_WINDOW", "1m"),
		AuthPasswordResetRateLimit:    envInt("AUTH_RATE_LIMIT_PASSWORD_RESET", 5),
		AuthPasswordResetRateWindow:   env("AUTH_RATE_LIMIT_PASSWORD_RESET_WINDOW", "15m"),
		AuthVerifyEmailRateLimit:      envInt("AUTH_RATE_LIMIT_VERIFY_EMAIL", 5),
		AuthVerifyEmailRateWindow:     env("AUTH_RATE_LIMIT_VERIFY_EMAIL_WINDOW", "15m"),
		AuthCookieMode:                envBool("AUTH_COOKIE_MODE", false),
		AuthCookieSecure:              envBool("AUTH_COOKIE_SECURE", false),
		AuthCookieDomain:              env("AUTH_COOKIE_DOMAIN", ""),
		AuthCookieSameSite:            strings.ToLower(env("AUTH_COOKIE_SAME_SITE", "lax")),
		AuthCSRFEnabled:               envBool("AUTH_CSRF_ENABLED", true),
		LogLevel:                      strings.ToLower(env("LOG_LEVEL", "info")),
		ReadTimeout:                   env("HTTP_READ_TIMEOUT", "10s"),
		WriteTimeout:                  env("HTTP_WRITE_TIMEOUT", "15s"),
		JobsEnabled:                   envBool("JOBS_ENABLED", true),
		CleanupInterval:               env("JOB_CLEANUP_INTERVAL", "1h"),
		EmailPollInterval:             env("JOB_EMAIL_POLL_INTERVAL", "5s"),
		EmailLockTTL:                  env("JOB_EMAIL_LOCK_TTL", "2m"),
		EmailBatchSize:                envInt("JOB_EMAIL_BATCH_SIZE", 20),
		EmailMaxAttempts:              envInt("JOB_EMAIL_MAX_ATTEMPTS", 5),
		EmailRetention:                env("JOB_EMAIL_RETENTION", "168h"),
		StatsRefreshInterval:          env("JOB_STATS_REFRESH_INTERVAL", "10m"),
		RecommendationRefreshInterval: env("JOB_RECOMMENDATIONS_REFRESH_INTERVAL", "15m"),
		RecommendationActivityWindow:  env("JOB_RECOMMENDATION_ACTIVITY_WINDOW", "168h"),
		RecommendationUserBatchSize:   envInt("JOB_RECOMMENDATION_USER_BATCH_SIZE", 200),
		RecommendationLimit:           envInt("JOB_RECOMMENDATION_LIMIT", 20),
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
	loginFailureWindow, err := time.ParseDuration(raw.LoginFailureWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_LOGIN_FAILURE_WINDOW: %w", err)
	}
	loginLockoutDuration, err := time.ParseDuration(raw.LoginLockoutDuration)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_LOGIN_LOCKOUT_DURATION: %w", err)
	}
	authRegisterRateWindow, err := time.ParseDuration(raw.AuthRegisterRateWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_RATE_LIMIT_REGISTER_WINDOW: %w", err)
	}
	authLoginRateWindow, err := time.ParseDuration(raw.AuthLoginRateWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_RATE_LIMIT_LOGIN_WINDOW: %w", err)
	}
	authRefreshRateWindow, err := time.ParseDuration(raw.AuthRefreshRateWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_RATE_LIMIT_REFRESH_WINDOW: %w", err)
	}
	authPasswordResetRateWindow, err := time.ParseDuration(raw.AuthPasswordResetRateWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_RATE_LIMIT_PASSWORD_RESET_WINDOW: %w", err)
	}
	authVerifyEmailRateWindow, err := time.ParseDuration(raw.AuthVerifyEmailRateWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_RATE_LIMIT_VERIFY_EMAIL_WINDOW: %w", err)
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
	if loginFailureWindow <= 0 || loginLockoutDuration <= 0 || authRegisterRateWindow <= 0 || authLoginRateWindow <= 0 || authRefreshRateWindow <= 0 || authPasswordResetRateWindow <= 0 || authVerifyEmailRateWindow <= 0 {
		return Config{}, fmt.Errorf("security windows and lockout duration must be greater than zero")
	}

	readTimeout, err := time.ParseDuration(raw.ReadTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := time.ParseDuration(raw.WriteTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_WRITE_TIMEOUT: %w", err)
	}
	cleanupInterval, err := time.ParseDuration(raw.CleanupInterval)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_CLEANUP_INTERVAL: %w", err)
	}
	emailPollInterval, err := time.ParseDuration(raw.EmailPollInterval)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_EMAIL_POLL_INTERVAL: %w", err)
	}
	emailLockTTL, err := time.ParseDuration(raw.EmailLockTTL)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_EMAIL_LOCK_TTL: %w", err)
	}
	emailRetention, err := time.ParseDuration(raw.EmailRetention)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_EMAIL_RETENTION: %w", err)
	}
	statsRefreshInterval, err := time.ParseDuration(raw.StatsRefreshInterval)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_STATS_REFRESH_INTERVAL: %w", err)
	}
	recommendationRefreshInterval, err := time.ParseDuration(raw.RecommendationRefreshInterval)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_RECOMMENDATIONS_REFRESH_INTERVAL: %w", err)
	}
	recommendationActivityWindow, err := time.ParseDuration(raw.RecommendationActivityWindow)
	if err != nil {
		return Config{}, fmt.Errorf("parse JOB_RECOMMENDATION_ACTIVITY_WINDOW: %w", err)
	}
	if cleanupInterval <= 0 || emailPollInterval <= 0 || emailLockTTL <= 0 || emailRetention <= 0 || statsRefreshInterval <= 0 || recommendationRefreshInterval <= 0 || recommendationActivityWindow <= 0 {
		return Config{}, fmt.Errorf("job intervals and retention must be greater than zero")
	}

	cfg := Config{
		Env:                           raw.Env,
		HTTPPort:                      raw.HTTPPort,
		DatabaseURL:                   raw.DatabaseURL,
		JWTSecret:                     raw.JWTSecret,
		AppBaseURL:                    raw.AppBaseURL,
		MailFrom:                      raw.MailFrom,
		AdminEmails:                   parseEmailList(raw.AdminEmails),
		AccessTokenTTL:                accessTTL,
		RefreshTokenTTL:               refreshTTL,
		EmailVerifyTTL:                emailVerifyTTL,
		PasswordResetTTL:              passwordResetTTL,
		LoginFailureWindow:            loginFailureWindow,
		LoginMaxAttempts:              raw.LoginMaxAttempts,
		LoginLockoutDuration:          loginLockoutDuration,
		AuthRegisterRateLimit:         raw.AuthRegisterRateLimit,
		AuthRegisterRateWindow:        authRegisterRateWindow,
		AuthLoginRateLimit:            raw.AuthLoginRateLimit,
		AuthLoginRateWindow:           authLoginRateWindow,
		AuthRefreshRateLimit:          raw.AuthRefreshRateLimit,
		AuthRefreshRateWindow:         authRefreshRateWindow,
		AuthPasswordResetRateLimit:    raw.AuthPasswordResetRateLimit,
		AuthPasswordResetRateWindow:   authPasswordResetRateWindow,
		AuthVerifyEmailRateLimit:      raw.AuthVerifyEmailRateLimit,
		AuthVerifyEmailRateWindow:     authVerifyEmailRateWindow,
		AuthCookieMode:                raw.AuthCookieMode,
		AuthCookieSecure:              raw.AuthCookieSecure,
		AuthCookieDomain:              strings.TrimSpace(raw.AuthCookieDomain),
		AuthCookieSameSite:            raw.AuthCookieSameSite,
		AuthCSRFEnabled:               raw.AuthCSRFEnabled,
		LogLevel:                      raw.LogLevel,
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		JobsEnabled:                   raw.JobsEnabled,
		CleanupInterval:               cleanupInterval,
		EmailPollInterval:             emailPollInterval,
		EmailLockTTL:                  emailLockTTL,
		EmailBatchSize:                raw.EmailBatchSize,
		EmailMaxAttempts:              raw.EmailMaxAttempts,
		EmailRetention:                emailRetention,
		StatsRefreshInterval:          statsRefreshInterval,
		RecommendationRefreshInterval: recommendationRefreshInterval,
		RecommendationActivityWindow:  recommendationActivityWindow,
		RecommendationUserBatchSize:   raw.RecommendationUserBatchSize,
		RecommendationLimit:           raw.RecommendationLimit,
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

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(env(key, ""))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
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
