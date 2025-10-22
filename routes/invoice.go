package routes

import (
	"github.com/gofiber/fiber/v2"
)

// Pour l'instant, on crée juste la fonction SetupInvoiceRoutes vide
func SetupInvoiceRoutes(app *fiber.App) {
	invoice := app.Group("/invoices")
	invoice.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "invoice route OK"})
	})
}
