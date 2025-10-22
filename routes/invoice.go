package routes

import (
	"math/rand"

	"carbon-saas/database"
	"carbon-saas/middleware"
	"carbon-saas/models"

	"github.com/gofiber/fiber/v2"
)

func SetupInvoiceRoutes(app *fiber.App) {
	invoice := app.Group("/invoices", middleware.JWTMiddleware)
	invoice.Post("/upload", uploadInvoice)
	invoice.Get("/", getInvoices)
}

func uploadInvoice(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Fichier requis"})
	}

	userID := c.Locals("user_id").(uint)

	// Simulation analyse (à améliorer avec OCR)
	amount := 100.0 + rand.Float64()*900.0
	co2 := amount * 0.3 // Facteur carbone simplifié

	inv := models.Invoice{
		UserID:       userID,
		FileName:     file.Filename,
		OriginalName: file.Filename,
		TotalAmount:  amount,
		CO2Estimate:  co2,
		TextPreview:  "Facture analysée",
	}

	database.DB.Create(&inv)

	return c.JSON(fiber.Map{
		"message":      "Facture uploadée",
		"co2_estimate": co2,
	})
}

func getInvoices(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var invoices []models.Invoice
	database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&invoices)
	return c.JSON(fiber.Map{"invoices": invoices})
}