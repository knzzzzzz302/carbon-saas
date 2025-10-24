package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"

	"carbon-saas/database"
	"carbon-saas/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("pas de .env trouvé")
	}

	database.ConnectDB()

	app := fiber.New()

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Routes API
	routes.SetupAuthRoutes(app)
	routes.SetupInvoiceRoutes(app)

	// Route racine : page de login
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./public/login.html")
	})

	// Route dashboard : accessible après connexion
	app.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.SendFile("./public/dashboard.html")
	})

	// Servir les autres fichiers statiques
	app.Static("/", "./public")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Println("🚀 Serveur sur http://localhost:" + port)
	log.Fatal(app.Listen(":" + port))
}