package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"carbon-saas/database"
	"carbon-saas/integrations/mistral"
	"carbon-saas/models"
)

const (
	defaultTimeout = 30 * time.Second
)

type Orchestrator struct {
	client *mistral.Client
}

type AnalyticsResult struct {
	Narrative       string             `json:"narrative"`
	KeyFindings     []string           `json:"key_findings"`
	Recommendations []string           `json:"recommendations"`
	Metrics         map[string]any     `json:"metrics"`
	Scopes          map[string]float64 `json:"scopes"`
	RawAI           any                `json:"raw_ai"`
}

type SuppliersResult struct {
	Narrative string         `json:"narrative"`
	Suppliers []SupplierInfo `json:"suppliers"`
	RawAI     any            `json:"raw_ai"`
}

type SupplierInfo struct {
	Name            string  `json:"name"`
	Spend           float64 `json:"spend"`
	CO2             float64 `json:"co2"`
	Priority        string  `json:"priority"`
	RecommendedStep string  `json:"recommended_step"`
}

type ChatResult struct {
	Message string `json:"message"`
	RawAI   any    `json:"raw_ai"`
}

var orchestrator *Orchestrator

func Init() {
	client, err := mistral.NewClientFromEnv()
	if err != nil {
		orchestrator = nil
		return
	}
	orchestrator = &Orchestrator{client: client}
}

func Get() *Orchestrator {
	return orchestrator
}

func (o *Orchestrator) IsReady() bool {
	return o != nil && o.client != nil
}

func (o *Orchestrator) GenerateAnalytics(ctx context.Context, tenantID uint) (AnalyticsResult, error) {
	var result AnalyticsResult
	if !o.IsReady() {
		return result, errors.New("Mistral non configuré")
	}

	var tenant models.Tenant
	if err := database.DB.First(&tenant, tenantID).Error; err != nil {
		return result, err
	}

	var invoices []models.Invoice
	if err := database.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").Limit(200).Find(&invoices).Error; err != nil {
		return result, err
	}

	facts := buildAnalyticsFacts(tenant, invoices)
	payload, _ := json.Marshal(facts)
	prompt := fmt.Sprintf(`Tu es un expert Greenly spécialisé en bilan carbone entreprise.
Analyse le JSON suivant et renvoie:
- Un résumé stratégique (ton directif, données chiffrées)
- 3 à 5 constats clés
- 3 recommandations priorisées

JSON: %s`, string(payload))

	resp, err := o.client.SendConversation(ctx, prompt)
	if err != nil {
		return result, err
	}

	result = AnalyticsResult{
		Narrative:       resp.FirstText(),
		KeyFindings:     facts.KeyFindings,
		Recommendations: facts.Recommendations,
		Metrics:         facts.Metrics,
		Scopes:          facts.Scopes,
		RawAI:           resp,
	}
	if len(result.KeyFindings) == 0 {
		result.KeyFindings = facts.DefaultFindings()
	}
	if len(result.Recommendations) == 0 {
		result.Recommendations = facts.DefaultRecs()
	}
	return result, nil
}

func (o *Orchestrator) GenerateSupplierInsights(ctx context.Context, tenantID uint) (SuppliersResult, error) {
	var result SuppliersResult
	if !o.IsReady() {
		return result, errors.New("Mistral non configuré")
	}

	var invoices []models.Invoice
	if err := database.DB.Where("tenant_id = ?", tenantID).Find(&invoices).Error; err != nil {
		return result, err
	}

	suppliers := groupSuppliers(invoices)
	payload, _ := json.Marshal(suppliers)
	prompt := fmt.Sprintf(`Agis comme responsable achats durables.
En te basant sur ces agrégats fournisseurs (JSON), produis:
- Liste structurée des fournisseurs avec spend, CO2, priorité(haute/moyenne/basse)
- Action recommandée par fournisseur.

JSON: %s`, string(payload))

	resp, err := o.client.SendConversation(ctx, prompt)
	if err != nil {
		return result, err
	}

	result = SuppliersResult{
		Narrative: resp.FirstText(),
		Suppliers: suppliers.ToSupplierInfo(),
		RawAI:     resp,
	}
	return result, nil
}

func (o *Orchestrator) ChatWithContext(ctx context.Context, tenantID uint, prompt string) (ChatResult, error) {
	var result ChatResult
	if strings.TrimSpace(prompt) == "" {
		return result, errors.New("prompt requis")
	}
	if !o.IsReady() {
		return result, errors.New("Mistral non configuré")
	}

	snapshot := buildChatSnapshot(tenantID)
	payload, _ := json.Marshal(snapshot)
	fullPrompt := fmt.Sprintf(`Tu es l'assistant climat principal. Contexte JSON: %s.
Réponds en français, ton expert mais accessible. Question: %s`, string(payload), prompt)

	resp, err := o.client.SendConversation(ctx, fullPrompt)
	if err != nil {
		return result, err
	}

	result = ChatResult{
		Message: resp.FirstText(),
		RawAI:   resp,
	}
	return result, nil
}

// --- Helpers

type analyticsFacts struct {
	TenantName      string             `json:"tenant_name"`
	Plan            string             `json:"plan"`
	InvoiceCount    int                `json:"invoice_count"`
	TotalSpend      float64            `json:"total_spend"`
	TotalCO2        float64            `json:"total_co2"`
	AverageCO2PerE  float64            `json:"avg_co2_per_euro"`
	LatestInvoices  []invoiceSnapshot  `json:"latest_invoices"`
	Scopes          map[string]float64 `json:"scopes"`
	KeyFindings     []string           `json:"-"`
	Recommendations []string           `json:"-"`
	Metrics         map[string]any     `json:"-"`
}

type invoiceSnapshot struct {
	Reference string  `json:"reference"`
	Supplier  string  `json:"supplier"`
	Amount    float64 `json:"amount"`
	CO2       float64 `json:"co2"`
}

func buildAnalyticsFacts(tenant models.Tenant, invoices []models.Invoice) analyticsFacts {
	var totalSpend, totalCO2 float64
	var latest []invoiceSnapshot
	for idx, inv := range invoices {
		totalSpend += inv.TotalAmount
		totalCO2 += inv.CO2Estimate
		if idx < 10 {
			latest = append(latest, invoiceSnapshot{
				Reference: inv.FileName,
				Supplier:  inv.OriginalName,
				Amount:    inv.TotalAmount,
				CO2:       inv.CO2Estimate,
			})
		}
	}
	facts := analyticsFacts{
		TenantName:     tenant.Name,
		Plan:           tenant.Plan,
		InvoiceCount:   len(invoices),
		TotalSpend:     totalSpend,
		TotalCO2:       totalCO2,
		AverageCO2PerE: safeDivide(totalCO2, totalSpend),
		LatestInvoices: latest,
		Scopes: map[string]float64{
			"scope1": totalCO2 * 0.25,
			"scope2": totalCO2 * 0.35,
			"scope3": totalCO2 * 0.40,
		},
		Metrics: map[string]any{
			"total_spend":      totalSpend,
			"total_co2":        totalCO2,
			"invoice_count":    len(invoices),
			"avg_co2_per_euro": safeDivide(totalCO2, totalSpend),
		},
	}
	return facts
}

func (a analyticsFacts) DefaultFindings() []string {
	return []string{
		fmt.Sprintf("Dépenses totales analysées: %.2f EUR", a.TotalSpend),
		fmt.Sprintf("Empreinte totale estimée: %.2f tCO2e", a.TotalCO2),
	}
}

func (a analyticsFacts) DefaultRecs() []string {
	return []string{
		"Prioriser un plan d'atténuation sur les fournisseurs les plus émetteurs",
		"Déployer un monitoring mensuel des scopes 1/2/3 via l'assistant IA",
	}
}

type supplierAggregate map[string]*SupplierInfo

func groupSuppliers(invoices []models.Invoice) supplierAggregate {
	agg := supplierAggregate{}
	for _, inv := range invoices {
		key := inv.OriginalName
		if key == "" {
			key = inv.FileName
		}
		info, ok := agg[key]
		if !ok {
			info = &SupplierInfo{Name: key}
			agg[key] = info
		}
		info.Spend += inv.TotalAmount
		info.CO2 += inv.CO2Estimate
	}
	for _, info := range agg {
		switch {
		case info.CO2 > 500:
			info.Priority = "haute"
		case info.CO2 > 200:
			info.Priority = "moyenne"
		default:
			info.Priority = "basse"
		}
		info.RecommendedStep = "Engager une trajectoire science-based"
	}
	return agg
}

func (s supplierAggregate) ToSupplierInfo() []SupplierInfo {
	out := make([]SupplierInfo, 0, len(s))
	for _, info := range s {
		out = append(out, *info)
	}
	return out
}

func buildChatSnapshot(tenantID uint) map[string]any {
	var tenant models.Tenant
	database.DB.First(&tenant, tenantID)
	var invoices []models.Invoice
	database.DB.Where("tenant_id = ?", tenantID).Order("created_at desc").Limit(50).Find(&invoices)
	facts := buildAnalyticsFacts(tenant, invoices)
	suppliers := groupSuppliers(invoices).ToSupplierInfo()
	return map[string]any{
		"tenant":    tenant.Name,
		"plan":      tenant.Plan,
		"metrics":   facts.Metrics,
		"scopes":    facts.Scopes,
		"suppliers": suppliers,
	}
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
