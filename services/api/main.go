package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Version simple pour démarrer : API HTTP, healthcheck, et bootstrap DB/auth.

func main() {
	// Chargement éventuel du .env (si présent)
	_ = loadEnvIfExists()

	cfg := LoadConfig()

	db, err := NewDB(cfg)
	if err != nil {
		log.Fatalf("échec connexion base de données: %v", err)
	}
	defer db.Close()

	router := gin.Default()

	// Middlewares globaux
	router.Use(CORSMiddleware())

	// Petit endpoint racine pour éviter les 404 sur "/"
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "carbonv2-api",
			"status":  "ok",
		})
	})

	router.GET("/health", HealthHandler(cfg))

	authHandler := NewAuthHandler(db, cfg)
	entriesHandler := NewEntriesHandler(db)
	carbonHandler := NewCarbonHandler(db)
	documentsHandler := NewDocumentsHandler(db)
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authHandler.SignUp)
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", AuthMiddleware(cfg, db), authHandler.Me)
		}

		tenants := api.Group("/tenants", AuthMiddleware(cfg, db))
		{
			tenants.POST("/:tenantId/entries", entriesHandler.CreateEntry)
			tenants.GET("/:tenantId/entries", entriesHandler.ListEntries)
			tenants.POST("/:tenantId/import", entriesHandler.ImportCSV)

			// Documents (factures, contrats énergie, etc.) liés à un tenant
			tenants.POST("/:tenantId/documents", documentsHandler.UploadDocument)
			tenants.GET("/:tenantId/documents", documentsHandler.ListDocuments)

			// Endpoints MVP carbone multi-tenant
			tenants.POST("/:tenantId/entries/:entryId/compute-emission", carbonHandler.ComputeEmissionForEntry)
			tenants.GET("/:tenantId/emissions/summary", carbonHandler.EmissionsSummary)
			tenants.GET("/:tenantId/emissions", carbonHandler.ListEmissions)
		}

		mlHandler := NewMLHandler(cfg)
		ml := api.Group("/ml")
		{
			ml.POST("/chat", mlHandler.Chat)
			ml.POST("/classify-transaction", mlHandler.ClassifyTransaction)
			ml.POST("/predict-trajectory", mlHandler.PredictTrajectory)
			ml.POST("/generate-report", mlHandler.GenerateReport)
		}
	}

	srv := NewHTTPServer(cfg, router)

	// Démarrage gracieux
	go func() {
		log.Printf("CarbonV2 API en écoute sur %s\n", cfg.HTTPAddr())
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("serveur arrêté: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Arrêt du serveur CarbonV2...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("arrêt forcé: %v", err)
	}
}
