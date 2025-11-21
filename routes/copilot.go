package routes

import (
	"context"
	"log"
	"time"

	"carbon-saas/integrations/mistral"
	"carbon-saas/middleware"

	"github.com/gofiber/fiber/v2"
)

var copilotClient *mistral.Client

func SetupCopilotRoutes(app *fiber.App) {
	client, err := mistral.NewClientFromEnv()
	if err != nil {
		log.Printf("⚠️ Mistral désactivé: %v", err)
	} else {
		copilotClient = client
	}

	group := app.Group("/copilot", middleware.JWTMiddleware)
	group.Post("/query", handleCopilotQuery)
	group.Post("/embed", handleCopilotEmbedding)
}

type copilotQueryPayload struct {
	Prompt string `json:"prompt"`
}

func handleCopilotQuery(c *fiber.Ctx) error {
	if copilotClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Mistral non configuré"})
	}

	var payload copilotQueryPayload
	if err := c.BodyParser(&payload); err != nil || payload.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Prompt requis"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
	defer cancel()

	resp, err := copilotClient.SendConversation(ctx, payload.Prompt)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"id":      resp.ID,
		"message": resp.FirstText(),
		"status":  resp.Status,
	})
}

type embedPayload struct {
	Text string `json:"text"`
}

func handleCopilotEmbedding(c *fiber.Ctx) error {
	if copilotClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Mistral non configuré"})
	}
	var payload embedPayload
	if err := c.BodyParser(&payload); err != nil || payload.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Texte requis"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
	defer cancel()

	embedding, err := copilotClient.CreateEmbedding(ctx, payload.Text)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"dimensions": len(embedding),
		"vector":     embedding,
	})
}
