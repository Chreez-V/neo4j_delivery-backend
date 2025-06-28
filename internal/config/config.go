package config

import (
    "os"
    "strconv" // Este es el paquete necesario para Atoi()
)
// getEnv obtiene una variable de entorno o un valor por defecto
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt obtiene una variable de entorno como entero
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

type Config struct {
	Port          int
	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string
}

func LoadConfig() *Config {
	return &Config{
		Port:          getEnvAsInt("PORT", 8080),
		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "12345678"),
	}
}
