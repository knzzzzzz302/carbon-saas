package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"carbon-saas/database"
	"carbon-saas/routes"
)

func main() {
	// Charge .env (si présent)
	if err := godotenv.Load(); err != nil {
		log.Println("pas de .env trouvé, utilisation variables d'environnement système")
	}

	// Connexion DB + migrations
	database.ConnectDB()

	app := fiber.New()

	// Routes : (les fichiers routes/* seront ajoutés ensuite)
	routes.SetupAuthRoutes(app)
	routes.SetupInvoiceRoutes(app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Println("🚀 Serveur lancé sur http://localhost:" + port)
	log.Fatal(app.Listen(":" + port))
}
