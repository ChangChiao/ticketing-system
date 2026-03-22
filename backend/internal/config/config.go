package config

import "os"

type Config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string

	// LINE Pay
	LinePayChannelID     string
	LinePayChannelSecret string
	LinePayBaseURL       string // sandbox or production
	AppBaseURL           string // our app's public URL
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ticketing?sslmode=disable"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		JWTSecret:            getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		LinePayChannelID:     getEnv("LINEPAY_CHANNEL_ID", ""),
		LinePayChannelSecret: getEnv("LINEPAY_CHANNEL_SECRET", ""),
		LinePayBaseURL:       getEnv("LINEPAY_BASE_URL", "https://sandbox-api-pay.line.me"),
		AppBaseURL:           getEnv("APP_BASE_URL", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
