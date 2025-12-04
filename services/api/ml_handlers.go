package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handlers HTTP pour exposer les fonctionnalités IA (Mistral).

type MLHandler struct {
	mistral *MistralClient
}

func NewMLHandler(cfg Config) *MLHandler {
	return &MLHandler{
		mistral: NewMistralClient(cfg),
	}
}

// Chat générique pour le chatbot d'accueil.
// POST /api/ml/chat
type chatRequest struct {
	Message string `json:"message" binding:"required"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

func (h *MLHandler) Chat(c *gin.Context) {
	if !h.mistral.enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Mistral non configuré côté serveur"})
		return
	}

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	out, err := h.mistral.invokeAgent(req.Message)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec appel Mistral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chatResponse{Reply: out})
}

type classifyTransactionRequest struct {
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
}

type classifyTransactionResponse struct {
	Category string `json:"category"`
	Scope    string `json:"scope"` // "1","2","3"
	Reason   string `json:"reason"`
	Raw      string `json:"raw"`
}

// POST /api/ml/classify-transaction
func (h *MLHandler) ClassifyTransaction(c *gin.Context) {
	if !h.mistral.enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Mistral non configuré côté serveur"})
		return
	}

	var req classifyTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	prompt := buildClassificationPrompt(req.Description, req.Amount, req.Currency)
	out, err := h.mistral.invokeAgent(prompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec appel Mistral", "details": err.Error()})
		return
	}

	// Pour l’instant on renvoie le texte brut et on laisse le frontend parser / afficher.
	c.JSON(http.StatusOK, classifyTransactionResponse{
		Category: "",
		Scope:    "",
		Reason:   "",
		Raw:      out,
	})
}

type predictTrajectoryRequest struct {
	// Liste simplifiée d’émissions mensuelles (tCO2e).
	History []float64 `json:"history" binding:"required"`
}

type predictTrajectoryResponse struct {
	Raw string `json:"raw"`
}

// POST /api/ml/predict-trajectory
func (h *MLHandler) PredictTrajectory(c *gin.Context) {
	if !h.mistral.enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Mistral non configuré côté serveur"})
		return
	}

	var req predictTrajectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	prompt := buildForecastPrompt(req.History)
	out, err := h.mistral.invokeAgent(prompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec appel Mistral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, predictTrajectoryResponse{Raw: out})
}

type generateReportRequest struct {
	SummaryData map[string]interface{} `json:"summary_data" binding:"required"`
}

type generateReportResponse struct {
	Raw string `json:"raw"`
}

// POST /api/ml/generate-report
func (h *MLHandler) GenerateReport(c *gin.Context) {
	if !h.mistral.enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Mistral non configuré côté serveur"})
		return
	}

	var req generateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalide", "details": err.Error()})
		return
	}

	prompt, err := buildReportPrompt(req.SummaryData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "summary_data invalide", "details": err.Error()})
		return
	}

	out, err := h.mistral.invokeAgent(prompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "échec appel Mistral", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, generateReportResponse{Raw: out})
}


