package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Config contient la configuration principale de l'API.
type Config struct {
	Env            string
	Port           string
	JWTSecret      string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	MistralAPIKey  string
	MistralAgentID string
}

// LoadConfig charge la configuration à partir des variables d'environnement.
func LoadConfig() Config {
	cfg := Config{
		Env:            getEnv("API_ENV", "development"),
		Port:           getEnv("API_PORT", "8080"),
		JWTSecret:      getEnv("API_JWT_SECRET", "changeme-super-secret"),
		DBHost:         getEnv("API_DB_HOST", "localhost"),
		DBPort:         getEnv("API_DB_PORT", "5432"),
		DBUser:         getEnv("API_DB_USER", "carbonv2"),
		DBPassword:     getEnv("API_DB_PASSWORD", "carbonv2_password"),
		DBName:         getEnv("API_DB_NAME", "carbonv2"),
		MistralAPIKey:  getEnv("MISTRAL_API_KEY", "PXML059c2QesiVDtc8VcBh4NjX6OZsJq"),
		MistralAgentID: getEnv("MISTRAL_AGENT_ID", "ag_019aa6e42967756f96ee8200155ff336"),
	}

	if cfg.JWTSecret == "" || cfg.JWTSecret == "changeme-super-secret" {
		log.Println("[AVERTISSEMENT] API_JWT_SECRET n'est pas configuré ou utilise la valeur par défaut. Ne pas utiliser en production.")
	}

	if cfg.MistralAPIKey == "" {
		log.Println("[INFO] MISTRAL_API_KEY n'est pas configuré. Les fonctionnalités IA seront désactivées.")
	}

	return cfg
}

func (c Config) HTTPAddr() string {
	return ":" + c.Port
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// NewHTTPServer crée un serveur HTTP configuré.
func NewHTTPServer(cfg Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:           cfg.HTTPAddr(),
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

// HealthHandler renvoie un handler Gin simple pour le healthcheck.
func HealthHandler(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "carbonv2-api",
			"env":     cfg.Env,
		})
	}
}

// loadEnvIfExists charge un fichier .env local s'il existe.
func loadEnvIfExists() error {
	if _, err := os.Stat(".env"); err == nil {
		return godotenv.Load()
	}
	return nil
}
