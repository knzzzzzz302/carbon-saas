package routes

import (
	"errors"
	"os"
	"time"

	"carbon-saas/config"
	"carbon-saas/database"
	"carbon-saas/models"
	"carbon-saas/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	Company  string `json:"company"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
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

	orgName := body.Company
	if orgName == "" {
		orgName = body.Name + " Org"
	}
	tenant := models.Tenant{
		Name:         orgName,
		Slug:         utils.GenerateSlug(orgName),
		Plan:         models.TenantPlanStarter,
		Status:       "active",
		FeatureFlags: datatypes.JSONMap{},
		Metadata:     datatypes.JSONMap{},
	}

	if err := database.DB.Create(&tenant).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur création tenant"})
	}

	user := models.User{
		Name:            body.Name,
		Email:           body.Email,
		Password:        hash,
		DefaultTenantID: tenant.ID,
		Locale:          body.Locale,
		Timezone:        body.Timezone,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur création utilisateur"})
	}

	membership := models.Membership{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     models.RoleAdmin,
		Status:   "active",
	}
	if err := database.DB.Create(&membership).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur création membership"})
	}

	// génération JWT
	t, err := issueSessionToken(user, membership)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Token invalide"})
	}

	return c.JSON(fiber.Map{
		"token":  t,
		"tenant": tenant,
		"role":   membership.Role,
	})
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

	membership, err := resolveDefaultMembership(user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	t, err := issueSessionToken(user, membership)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Token invalide"})
	}

	return c.JSON(fiber.Map{
		"token":  t,
		"role":   membership.Role,
		"tenant": membership.Tenant,
	})
}

func issueSessionToken(user models.User, membership models.Membership) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID,
		"tenant_id": membership.TenantID,
		"role":      membership.Role,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
	})
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = config.DefaultJWTSecret
	}
	return token.SignedString([]byte(secret))
}

func resolveDefaultMembership(user models.User) (models.Membership, error) {
	var membership models.Membership
	query := database.DB.Preload("Tenant")
	if user.DefaultTenantID != 0 {
		query = query.Where("tenant_id = ? AND user_id = ?", user.DefaultTenantID, user.ID)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return membership, errors.New("Aucun tenant associé")
		}
		return membership, err
	}
	return membership, nil
}
