package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	db  *pgxpool.Pool
	cfg Config
}

func NewAuthHandler(db *pgxpool.Pool, cfg Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type signUpRequest struct {
	TenantName string `json:"tenant_name" binding:"required"`
	Siret      string `json:"siret" binding:"omitempty"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req signUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
		return
	}
	defer tx.Rollback(ctx)

	var tenantID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO tenants (name, siret, plan) VALUES ($1, $2, $3) RETURNING id`,
		req.TenantName, req.Siret, "free",
	).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de créer le tenant"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de sécurité"})
		return
	}

	var userID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO users (tenant_id, email, role, password_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, strings.ToLower(req.Email), "admin", string(passwordHash),
	).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de créer l'utilisateur"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur de validation"})
		return
	}

	token, err := h.createToken(userID, tenantID, "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de générer le token"})
		return
	}

	c.JSON(http.StatusCreated, authResponse{Token: token})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var (
		userID       int64
		tenantID     int64
		role         string
		passwordHash string
	)

	err := h.db.QueryRow(ctx,
		`SELECT id, tenant_id, role, password_hash FROM users WHERE email = $1`,
		strings.ToLower(req.Email),
	).Scan(&userID, &tenantID, &role, &passwordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identifiants invalides"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identifiants invalides"})
		return
	}

	token, err := h.createToken(userID, tenantID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de générer le token"})
		return
	}

	c.JSON(http.StatusOK, authResponse{Token: token})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) createToken(userID, tenantID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID,
		"tenant_id": tenantID,
		"role":      role,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"iss":       "carbonv2-api",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}
