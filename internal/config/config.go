package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	DBDSN                 string
	KafkaBrokers          string
	KafkaTopicSubmissions string
	RedisAddr             string
	HealthPort            string
}

// LoadConfig loads configuration from environment variables or .env file
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env file not found, reading system variables")
	}
	return &Config{
		Port:                  getEnv("SERVER_PORT", ":8080"),
		DBDSN:                 getEnv("DB_DSN", ""),
		KafkaBrokers:          getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopicSubmissions: getEnv("KAFKA_TOPIC_SUBMISSIONS", "submissions.pending"),
		RedisAddr:             getEnv("REDIS_ADDR", "localhost:6379"),
		HealthPort:            getEnv("HEALTH_PORT", ":9090"),
	}
}

// Helper function: gets value from OS or returns default
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
