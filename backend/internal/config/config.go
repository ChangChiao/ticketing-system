package config

import "os"

type Config struct {
	Port          string
	ServiceRole   string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string

	// LINE Pay
	LinePayChannelID     string
	LinePayChannelSecret string
	LinePayBaseURL       string // sandbox or production
	AppBaseURL           string // our app's public URL

	// Security
	TurnstileSecretKey string // Cloudflare Turnstile CAPTCHA
	RequestSignSecret  string // HMAC request signature secret
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		ServiceRole:          getEnv("SERVICE_ROLE", "all"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		JWTSecret:            getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		LinePayChannelID:     getEnv("LINEPAY_CHANNEL_ID", ""),
		LinePayChannelSecret: getEnv("LINEPAY_CHANNEL_SECRET", ""),
		LinePayBaseURL:       getEnv("LINEPAY_BASE_URL", "https://sandbox-api-pay.line.me"),
		AppBaseURL:           getEnv("APP_BASE_URL", "http://localhost:3000"),
		TurnstileSecretKey:   getEnv("TURNSTILE_SECRET_KEY", ""),
		RequestSignSecret:    getEnv("REQUEST_SIGN_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
