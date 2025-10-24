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

    // Routes API (AVANT les routes statiques)
    routes.SetupAuthRoutes(app)
    routes.SetupInvoiceRoutes(app)

    // Servir les fichiers statiques depuis le dossier public
    app.Static("/", "./public")

    // Route par défaut (doit être APRÈS Static)
    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendFile("./public/login.html")
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "3030"
    }
    log.Println("🚀 Serveur sur http://localhost:" + port)
    log.Fatal(app.Listen(":" + port))
}