package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Non autorisé"})
	}

	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Token invalide"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Claims invalides"})
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Utilisateur manquant"})
	}
	tenantID, ok := claims["tenant_id"].(float64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Tenant manquant"})
	}
	role, _ := claims["role"].(string)

	c.Locals("user_id", uint(userID))
	c.Locals("tenant_id", uint(tenantID))
	c.Locals("role", role)
	return c.Next()
}
