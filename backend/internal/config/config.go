package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	APIKey             string
	MongoURI           string
	MongoDB            string
	VictoriaLogsURL    string
	DockerHost         string
	MaxConcurrentTasks string
	GVisorRuntime      string
}

func Load() (*Config, error) {
	// .env es opcional — en producción las vars vienen del entorno
	_ = godotenv.Load()

	return &Config{
		Port:               getEnv("PORT", "8080"),
		APIKey:             getEnv("API_KEY", ""),
		MongoURI:           getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:            getEnv("MONGO_DB", "task_orchestrator"),
		VictoriaLogsURL:    getEnv("VICTORIA_LOGS_URL", "http://localhost:9428"),
		DockerHost:         getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		MaxConcurrentTasks: getEnv("MAX_CONCURRENT_TASKS", "10"),
		GVisorRuntime:      getEnv("GVISOR_RUNTIME", "runsc"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
