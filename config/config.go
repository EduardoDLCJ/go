package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	GinMode          string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	DBChannelBinding string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration
	AllowedOrigins   []string
	RateLimitAuth    int
	RateLimitWrite   int
	RateLimitRead    int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not load .env file: %v", err)
	}

	return &Config{
		Port:             getEnv("PORT", "10000"),
		GinMode:          getEnv("GIN_MODE", gin.DebugMode),
		DBHost:           getEnv("PGHOST", ""),
		DBPort:           getEnv("PGPORT", "5432"),
		DBUser:           getEnv("PGUSER", ""),
		DBPassword:       getEnv("PGPASSWORD", ""),
		DBName:           getEnv("PGDATABASE", ""),
		DBSSLMode:        getEnv("PGSSLMODE", "require"),
		DBChannelBinding: getEnv("PGCHANNELBINDING", "require"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTAccessExpiry:  getDurationEnv("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry: getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		AllowedOrigins:   getCSVEnv("ALLOWED_ORIGINS", []string{"*"}),
		RateLimitAuth:    getIntEnv("RATE_LIMIT_AUTH", 5),
		RateLimitWrite:   getIntEnv("RATE_LIMIT_WRITE", 10),
		RateLimitRead:    getIntEnv("RATE_LIMIT_READ", 60),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&channel_binding=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
		c.DBChannelBinding,
	)
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("warning: invalid duration for %s=%q, using %s", key, value, fallback)
		return fallback
	}
	return duration
}

func getIntEnv(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("warning: invalid integer for %s=%q, using %d", key, value, fallback)
		return fallback
	}
	return number
}

func getCSVEnv(key string, fallback []string) []string {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}
