package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DB       DBConfig
	JWT      JWTConfig
	Redis    RedisConfig
	HTTP     HTTPConfig
	Telegram TelegramConfig
	Notifier NotifierConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	OTPTTL     time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type HTTPConfig struct {
	Port               string
	AllowedOrigins     []string
	TrustProxy         bool
	RateLimitPerMinute int64
}

type NotifierConfig struct {
	Timeout time.Duration
	SMTP    SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

type TelegramConfig struct {
	BotToken    string
	WebAppURL   string
	PollTimeout time.Duration
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Load() (*Config, error) {
	accessTTL, err := getEnvDuration("ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := getEnvDuration("REFRESH_TTL", 720*time.Hour)
	if err != nil {
		return nil, err
	}
	otpTTL, err := getEnvDuration("OTP_TTL", 5*time.Minute)
	if err != nil {
		return nil, err
	}
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	rateLimit, err := getEnvInt("RATE_LIMIT_PER_MINUTE", 120)
	if err != nil {
		return nil, err
	}

	notifierTimeout, err := getEnvDuration("NOTIFIER_TIMEOUT", 15*time.Second)
	if err != nil {
		return nil, err
	}

	tgPollTimeout, err := getEnvDuration("TELEGRAM_POLL_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", ""),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
			OTPTTL:     otpTTL,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		HTTP: HTTPConfig{
			Port:               getEnv("HTTP_PORT", "8087"),
			AllowedOrigins:     getEnvList("CORS_ALLOWED_ORIGINS"),
			TrustProxy:         getEnv("TRUST_PROXY", "") == "true",
			RateLimitPerMinute: int64(rateLimit),
		},
		Notifier: NotifierConfig{
			Timeout: notifierTimeout,
			SMTP: SMTPConfig{
				Host:     getEnv("SMTP_HOST", ""),
				Port:     getEnv("SMTP_PORT", "587"),
				Username: getEnv("SMTP_USERNAME", ""),
				Password: getEnv("SMTP_PASSWORD", ""),
				From:     getEnv("SMTP_FROM", ""),
				FromName: getEnv("SMTP_FROM_NAME", ""),
			},
		},
		Telegram: TelegramConfig{
			BotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
			WebAppURL:   getEnv("WEB_APP_URL", ""),
			PollTimeout: tgPollTimeout,
		},
	}

	if cfg.DB.User == "" || cfg.DB.Password == "" || cfg.DB.Name == "" {
		return nil, errors.New("config: missing required database credentials")
	}

	if len(cfg.JWT.Secret) < 32 {
		return nil, errors.New("config: JWT secret must be at least 32 characters")
	}

	return cfg, nil
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.Name,
	)
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("config: %s: invalid duration for %q: %w", key, value, err)
	}
	return d, nil
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("config: %s: invalid integer for %q: %w", key, value, err)
	}
	return n, nil
}

func getEnvList(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}

	var out []string
	for _, part := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}
