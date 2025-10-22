package routes

import (
	"os"
	"time"

	"carbon-saas/database"
	"carbon-saas/models"
	"carbon-saas/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func SetupAuthRoutes(app *fiber.App) {
	auth := app.Group("/auth")
	auth.Post("/register", register)
	auth.Post("/login", login)
}

type authPayload struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func register(c *fiber.Ctx) error {
	var body authPayload
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload invalide"})
	}

	// vérifier si email déjà existant
	var existing models.User
	database.DB.Where("email = ?", body.Email).First(&existing)
	if existing.ID != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email déjà enregistré"})
	}

	hash, err := utils.HashPassword(body.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Impossible de hasher le mot de passe"})
	}

	user := models.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: hash,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur création utilisateur"})
	}

	// génération JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	t, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	return c.JSON(fiber.Map{"token": t})
}

func login(c *fiber.Ctx) error {
	var body authPayload
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload invalide"})
	}

	var user models.User
	database.DB.Where("email = ?", body.Email).First(&user)
	if user.ID == 0 || !utils.CheckPassword(user.Password, body.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email ou mot de passe invalide"})
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	t, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	return c.JSON(fiber.Map{"token": t})
}
