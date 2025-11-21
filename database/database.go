package database

import (
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"carbon-saas/models"
)

var DB *gorm.DB

func ConnectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "carbon.db"
	}

	var dialector gorm.Dialector
	normalized := strings.ToLower(dsn)
	isPostgres := strings.HasPrefix(normalized, "postgres://") ||
		strings.HasPrefix(normalized, "postgresql://") ||
		strings.Contains(normalized, "host=") ||
		strings.Contains(normalized, "user=") ||
		strings.Contains(normalized, "dbname=")

	if isPostgres {
		dialector = postgres.Open(dsn)
	} else {
		dialector = sqlite.Open(dsn)
	}

	database, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal("Erreur connexion DB:", err)
	}

	if err := database.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Membership{},
		&models.ServiceAccount{},
		&models.Invoice{},
	); err != nil {
		log.Fatal("Erreur migration:", err)
	}

	DB = database
	log.Println("📦 DB connectée et migrée sur", dsn)
}
