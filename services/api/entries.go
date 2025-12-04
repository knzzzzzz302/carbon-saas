package main

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EntriesHandler struct {
	db *pgxpool.Pool
}

func NewEntriesHandler(db *pgxpool.Pool) *EntriesHandler {
	return &EntriesHandler{db: db}
}

type createEntryRequest struct {
	Type     string            `json:"type" binding:"required"`
	Amount   float64           `json:"amount" binding:"required"`
	Currency string            `json:"currency" binding:"required"`
	Date     string            `json:"date" binding:"required"` // YYYY-MM-DD
	Category string            `json:"category"`
	Source   string            `json:"source"`
	Metadata map[string]string `json:"metadata"`
}

// POST /api/tenants/:tenantId/entries
func (h *EntriesHandler) CreateEntry(c *gin.Context) {
	claimsVal, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}
	claims := claimsVal.(jwt.MapClaims)

	pathTenant := c.Param("tenantId")
	tenantIDFromToken := claims["tenant_id"]
	if pathTenant != toStringID(tenantIDFromToken) {
		c.JSON(http.StatusForbidden, gin.H{"error": "accès interdit à ce tenant"})
		return
	}

	var req createEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	parsedDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date invalide, format attendu YYYY-MM-DD"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Convertir tenantIDFromToken en int64 (peut être float64 depuis JWT)
	var tenantIDInt int64
	switch v := tenantIDFromToken.(type) {
	case float64:
		tenantIDInt = int64(v)
	case int64:
		tenantIDInt = v
	case int:
		tenantIDInt = int64(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "format tenant_id invalide"})
		return
	}

	var entryID int64
	err = h.db.QueryRow(ctx,
		`INSERT INTO entries (tenant_id, type, amount, currency, date, category, source, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,COALESCE($8::jsonb, '{}'::jsonb))
		 RETURNING id`,
		tenantIDInt,
		req.Type,
		req.Amount,
		req.Currency,
		parsedDate,
		req.Category,
		req.Source,
		toJSONB(req.Metadata),
	).Scan(&entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de créer l'entrée", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": entryID})
}

// GET /api/tenants/:tenantId/entries
func (h *EntriesHandler) ListEntries(c *gin.Context) {
	claimsVal, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}
	claims := claimsVal.(map[string]interface{})

	pathTenant := c.Param("tenantId")
	tenantIDFromToken := claims["tenant_id"]
	if pathTenant != toStringID(tenantIDFromToken) {
		c.JSON(http.StatusForbidden, gin.H{"error": "accès interdit à ce tenant"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Convertir tenantIDFromToken en int64
	var tenantIDInt int64
	switch v := tenantIDFromToken.(type) {
	case float64:
		tenantIDInt = int64(v)
	case int64:
		tenantIDInt = v
	case int:
		tenantIDInt = int64(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "format tenant_id invalide"})
		return
	}

	rows, err := h.db.Query(ctx,
		`SELECT id, type, amount, currency, date, category, source
		 FROM entries
		 WHERE tenant_id = $1
		 ORDER BY date DESC, created_at DESC
		 LIMIT 100`,
		tenantIDInt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la récupération des entrées"})
		return
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var e Entry
		var category, source *string
		err := rows.Scan(&e.ID, &e.Type, &e.Amount, &e.Currency, &e.Date, &category, &source)
		if err != nil {
			continue
		}
		entry := map[string]interface{}{
			"id":       e.ID,
			"type":     e.Type,
			"amount":   e.Amount,
			"currency": e.Currency,
			"date":     e.Date.Format("2006-01-02"),
		}
		if category != nil {
			entry["category"] = *category
		}
		if source != nil {
			entry["source"] = *source
		}
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, entries)
}

// POST /api/tenants/:tenantId/import (CSV simple)
// Format attendu (en-têtes) : type,amount,currency,date,category,source
func (h *EntriesHandler) ImportCSV(c *gin.Context) {
	claimsVal, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "non authentifié"})
		return
	}
	claims := claimsVal.(jwt.MapClaims)

	pathTenant := c.Param("tenantId")
	tenantIDFromToken := claims["tenant_id"]
	if pathTenant != toStringID(tenantIDFromToken) {
		c.JSON(http.StatusForbidden, gin.H{"error": "accès interdit à ce tenant"})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fichier CSV manquant"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV invalide ou vide"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur interne"})
		return
	}
	defer tx.Rollback(ctx)

	inserted := 0
	for i, row := range records {
		if i == 0 {
			continue // en-tête
		}
		if len(row) < 4 {
			continue
		}
		amount, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			continue
		}
		dateVal, err := time.Parse("2006-01-02", row[3])
		if err != nil {
			continue
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO entries (tenant_id, type, amount, currency, date, category, source)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			tenantIDFromToken,
			row[0],
			amount,
			row[2],
			dateVal,
			valueOrEmpty(row, 4),
			valueOrEmpty(row, 5),
		)
		if err == nil {
			inserted++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "échec de l'import"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"inserted": inserted})
}

func valueOrEmpty(row []string, idx int) string {
	if len(row) > idx {
		return row[idx]
	}
	return ""
}
