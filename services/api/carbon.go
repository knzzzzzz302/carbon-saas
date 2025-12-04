package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CarbonHandler regroupe les endpoints liés au calcul et à la synthèse carbone.
type CarbonHandler struct {
	db *pgxpool.Pool
}

func NewCarbonHandler(db *pgxpool.Pool) *CarbonHandler {
	return &CarbonHandler{db: db}
}

// Règles ultra-simples de facteur d'émission pour le MVP.
// On travaille en kgCO2e / EUR puis on convertit en tCO2e.
type emissionRule struct {
	FactorKgPerEUR float64
	Scope          string
}

// getRule retourne une règle approximative en fonction de la catégorie ou du type.
func getRule(category, entryType string) emissionRule {
	key := strings.ToLower(strings.TrimSpace(category))
	t := strings.ToLower(strings.TrimSpace(entryType))

	// Quelques exemples très grossiers pour le MVP.
	switch {
	case strings.Contains(key, "avion") || strings.Contains(t, "flight"):
		return emissionRule{FactorKgPerEUR: 0.6, Scope: "3"}
	case strings.Contains(key, "train"):
		return emissionRule{FactorKgPerEUR: 0.1, Scope: "3"}
	case strings.Contains(key, "élec") || strings.Contains(key, "electric") || strings.Contains(t, "energy"):
		return emissionRule{FactorKgPerEUR: 0.3, Scope: "2"}
	case strings.Contains(key, "fuel") || strings.Contains(key, "carburant") || strings.Contains(t, "fuel"):
		return emissionRule{FactorKgPerEUR: 0.5, Scope: "1"}
	default:
		// fallback très conservateur scope 3 générique
		return emissionRule{FactorKgPerEUR: 0.25, Scope: "3"}
	}
}

type computeEmissionResponse struct {
	EntryID    int64   `json:"entry_id"`
	EmissionID int64   `json:"emission_id"`
	Scope      string  `json:"scope"`
	TCO2e      float64 `json:"tco2e"`
}

// POST /api/tenants/:tenantId/entries/:entryId/compute-emission
// Calcule une émission simple à partir du montant et de la catégorie, et la stocke dans la table emissions.
func (h *CarbonHandler) ComputeEmissionForEntry(c *gin.Context) {
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

	entryIDStr := c.Param("entryId")
	if entryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entryId manquant"})
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

	// Récupère l'entrée et vérifie qu'elle appartient bien au tenant.
	var e Entry
	row := h.db.QueryRow(ctx,
		`SELECT id, tenant_id, type, amount, currency, date, category, source, created_at
		 FROM entries
		 WHERE id = $1 AND tenant_id = $2`,
		entryIDStr,
		tenantIDInt,
	)

	var category, source *string
	err := row.Scan(
		&e.ID,
		&e.TenantID,
		&e.Type,
		&e.Amount,
		&e.Currency,
		&e.Date,
		&category,
		&source,
		&e.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "entrée non trouvée"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la récupération de l'entrée"})
		return
	}

	catVal := ""
	if category != nil {
		catVal = *category
	}
	rule := getRule(catVal, e.Type)

	// Conversion très grossière en tCO2e.
	kg := e.Amount * rule.FactorKgPerEUR
	tco2e := kg / 1000.0

	var emissionID int64
	err = h.db.QueryRow(ctx,
		`INSERT INTO emissions (entry_id, tenant_id, scope, tco2e, methodology_version)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		e.ID,
		e.TenantID,
		rule.Scope,
		tco2e,
		"mvp-simple-v1",
	).Scan(&emissionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible d'enregistrer l'émission"})
		return
	}

	c.JSON(http.StatusCreated, computeEmissionResponse{
		EntryID:    e.ID,
		EmissionID: emissionID,
		Scope:      rule.Scope,
		TCO2e:      tco2e,
	})
}

type emissionsSummaryResponse struct {
	TenantID       string             `json:"tenant_id"`
	TotalTCO2e     float64            `json:"total_tco2e"`
	ByScope        map[string]float64 `json:"by_scope"`
	EntriesCount   int64              `json:"entries_count"`
	EmissionsCount int64              `json:"emissions_count"`
}

// GET /api/tenants/:tenantId/emissions/summary
// Retourne un petit résumé multi-tenant des émissions calculées.
func (h *CarbonHandler) EmissionsSummary(c *gin.Context) {
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

	// Agrégation par scope.
	rows, err := h.db.Query(ctx,
		`SELECT scope, COALESCE(SUM(tco2e), 0) 
		 FROM emissions 
		 WHERE tenant_id = $1 
		 GROUP BY scope`,
		tenantIDInt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la récupération des émissions"})
		return
	}
	defer rows.Close()

	byScope := make(map[string]float64)
	var total float64
	for rows.Next() {
		var scope string
		var sum float64
		if err := rows.Scan(&scope, &sum); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de l'agrégation des émissions"})
			return
		}
		byScope[scope] = sum
		total += sum
	}

	// Stat basique sur le nombre d'entrées et d'émissions.
	var entriesCount, emissionsCount int64
	if err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM entries WHERE tenant_id = $1`,
		tenantIDInt,
	).Scan(&entriesCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors du comptage des entrées"})
		return
	}

	if err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM emissions WHERE tenant_id = $1`,
		tenantIDInt,
	).Scan(&emissionsCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors du comptage des émissions"})
		return
	}

	c.JSON(http.StatusOK, emissionsSummaryResponse{
		TenantID:       pathTenant,
		TotalTCO2e:     total,
		ByScope:        byScope,
		EntriesCount:   entriesCount,
		EmissionsCount: emissionsCount,
	})
}

// GET /api/tenants/:tenantId/emissions
func (h *CarbonHandler) ListEmissions(c *gin.Context) {
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
		`SELECT id, entry_id, scope, tco2e, computed_at
		 FROM emissions
		 WHERE tenant_id = $1
		 ORDER BY computed_at DESC
		 LIMIT 100`,
		tenantIDInt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la récupération des émissions"})
		return
	}
	defer rows.Close()

	var emissions []map[string]interface{}
	for rows.Next() {
		var id, entryID int64
		var scope string
		var tco2e float64
		var computedAt time.Time
		err := rows.Scan(&id, &entryID, &scope, &tco2e, &computedAt)
		if err != nil {
			continue
		}
		emissions = append(emissions, map[string]interface{}{
			"id":          id,
			"entry_id":    entryID,
			"scope":       scope,
			"tco2e":       tco2e,
			"computed_at": computedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, emissions)
}
