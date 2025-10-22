package database

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"carbon-saas/models"
)

var DB *gorm.DB

func ConnectDB() {
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "carbon.db"
	}

	database, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Erreur connexion DB:", err)
	}

	// AutoMigrate : crée/ajuste les tables selon les modèles
	if err := database.AutoMigrate(&models.User{}, &models.Invoice{}); err != nil {
		log.Fatal("Erreur migration:", err)
	}

	DB = database
	log.Println("📦 DB connectée et migrée")
}
