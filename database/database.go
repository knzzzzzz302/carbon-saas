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
	var dialector gorm.Dialector

	switch {
	case strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://"):
		dialector = postgres.Open(dsn)
	case dsn != "":
		// Assume postgres DSN even without schema prefix
		dialector = postgres.Open(dsn)
	default:
		dbPath := "carbon.db"
		dialector = sqlite.Open(dbPath)
		dsn = dbPath
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
