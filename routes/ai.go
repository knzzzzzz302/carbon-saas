package routes

import (
	"context"
	"time"

	"carbon-saas/middleware"
	"carbon-saas/services/ai"

	"github.com/gofiber/fiber/v2"
)

func SetupAIRoutes(app *fiber.App) {
	ai.Init()
	group := app.Group("/ai", middleware.JWTMiddleware)

	group.Get("/status", func(c *fiber.Ctx) error {
		orch := ai.Get()
		return c.JSON(fiber.Map{
			"ready": orch != nil && orch.IsReady(),
		})
	})

	group.Get("/analytics", handleAIAnalytics)
	group.Get("/suppliers", handleAISuppliers)
	group.Post("/chat", handleAIChat)
}

func handleAIAnalytics(c *fiber.Ctx) error {
	orch := ai.Get()
	if orch == nil || !orch.IsReady() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "IA indisponible"})
	}
	tenantID := c.Locals("tenant_id").(uint)
	ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
	defer cancel()
	res, err := orch.GenerateAnalytics(ctx, tenantID)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func handleAISuppliers(c *fiber.Ctx) error {
	orch := ai.Get()
	if orch == nil || !orch.IsReady() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "IA indisponible"})
	}
	tenantID := c.Locals("tenant_id").(uint)
	ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
	defer cancel()
	res, err := orch.GenerateSupplierInsights(ctx, tenantID)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

type chatPayload struct {
	Prompt string `json:"prompt"`
}

func handleAIChat(c *fiber.Ctx) error {
	orch := ai.Get()
	if orch == nil || !orch.IsReady() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "IA indisponible"})
	}
	var payload chatPayload
	if err := c.BodyParser(&payload); err != nil || payload.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Prompt requis"})
	}
	tenantID := c.Locals("tenant_id").(uint)
	ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
	defer cancel()
	res, err := orch.ChatWithContext(ctx, tenantID, payload.Prompt)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}
