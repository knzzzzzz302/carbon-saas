package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DocumentsHandler gère l'upload et la liste des documents (factures, PDF, etc.).
type DocumentsHandler struct {
	db *pgxpool.Pool
}

func NewDocumentsHandler(db *pgxpool.Pool) *DocumentsHandler {
	return &DocumentsHandler{db: db}
}

// POST /api/tenants/:tenantId/documents
// Upload d'un document lié au tenant (PDF de facture EDF, etc.).
func (h *DocumentsHandler) UploadDocument(c *gin.Context) {
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

	// Conversion tenant_id en int64
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

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fichier manquant", "details": err.Error()})
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fichier vide"})
		return
	}

	// Très simple filtrage de type mime : on accepte surtout les PDF et images.
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(make([]byte, 512))
	}
	if !isAllowedMime(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type de fichier non supporté", "details": contentType})
		return
	}

	// Répertoire de stockage local (simple pour le MVP).
	baseDir := "uploads"
	tenantDir := filepath.Join(baseDir, "tenant_"+pathTenant)
	if err := os.MkdirAll(tenantDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de créer le répertoire de stockage"})
		return
	}

	safeName := sanitizeFilename(header.Filename)
	filename := time.Now().Format("20060102-150405") + "-" + safeName
	fullPath := filepath.Join(tenantDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible de sauvegarder le fichier"})
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la copie du fichier"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var docID int64
	err = h.db.QueryRow(ctx,
		`INSERT INTO documents (tenant_id, original_name, mime_type, size_bytes, storage_path)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id`,
		tenantIDInt,
		header.Filename,
		contentType,
		written,
		fullPath,
	).Scan(&docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "impossible d'enregistrer le document", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            docID,
		"original_name": header.Filename,
		"mime_type":     contentType,
		"size_bytes":    written,
	})
}

// GET /api/tenants/:tenantId/documents
// Liste des documents déjà importés pour alimenter le Bilan Carbone.
func (h *DocumentsHandler) ListDocuments(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx,
		`SELECT id, original_name, mime_type, size_bytes, created_at
		 FROM documents
		 WHERE tenant_id = $1
		 ORDER BY created_at DESC
		 LIMIT 100`,
		tenantIDInt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erreur lors de la récupération des documents"})
		return
	}
	defer rows.Close()

	type docItem struct {
		ID           int64  `json:"id"`
		OriginalName string `json:"original_name"`
		MimeType     string `json:"mime_type"`
		SizeBytes    int64  `json:"size_bytes"`
		CreatedAt    string `json:"created_at"`
	}

	var docs []docItem
	for rows.Next() {
		var d docItem
		var created time.Time
		if err := rows.Scan(&d.ID, &d.OriginalName, &d.MimeType, &d.SizeBytes, &created); err != nil {
			continue
		}
		d.CreatedAt = created.Format(time.RFC3339)
		docs = append(docs, d)
	}

	c.JSON(http.StatusOK, docs)
}

// isAllowedMime filtre quelques types simples pour l'upload.
func isAllowedMime(m string) bool {
	if m == "" {
		return false
	}
	m = strings.ToLower(m)
	if strings.HasPrefix(m, "application/pdf") {
		return true
	}
	if strings.HasPrefix(m, "image/") {
		return true
	}
	if m == "text/csv" || strings.Contains(m, "excel") {
		return true
	}
	return false
}

// sanitizeFilename nettoie un nom de fichier pour éviter les caractères problématiques.
func sanitizeFilename(name string) string {
	if name == "" {
		return "document"
	}
	name = filepath.Base(name)
	// remplace les espaces et caractères douteux par des tirets bas
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}


