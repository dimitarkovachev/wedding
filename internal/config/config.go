package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	AdminPort      string
	DBPath         string
	SeedFile       string
	WebDir         string
	RateLimitRPS   float64
	RateLimitBurst int
	GinMode        string
}

func Load() *Config {
	return &Config{
		Port:           envOrDefault("PORT", "8080"),
		AdminPort:      envOrDefault("ADMIN_PORT", "9090"),
		DBPath:         envOrDefault("DB_PATH", "/data/wedding.db"),
		SeedFile:       os.Getenv("SEED_FILE"),
		WebDir:         envOrDefault("WEB_DIR", "web"),
		RateLimitRPS:   envOrDefaultFloat("RATE_LIMIT_RPS", 1),
		RateLimitBurst: envOrDefaultInt("RATE_LIMIT_BURST", 10),
		GinMode:        envOrDefault("GIN_MODE", "release"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func envOrDefaultFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
