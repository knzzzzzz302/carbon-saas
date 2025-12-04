package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildClassificationPrompt(desc string, amount float64, currency string) string {
	return fmt.Sprintf(`Tu es un expert en comptabilité carbone.
Classe la dépense suivante :
- Description : %s
- Montant : %.2f %s

Objectif :
- Identifier une catégorie carbone claire (par exemple : "Transport - Avion", "Numérique - Cloud", "Énergie - Électricité").
- Déterminer le scope principal (1, 2 ou 3).
- Expliquer en quelques phrases la logique de classification.

Réponds dans un format texte structuré et lisible pour un humain.`, desc, amount, strings.ToUpper(currency))
}

func buildForecastPrompt(history []float64) string {
	// On encode l'historique en JSON pour le fournir au modèle.
	data, _ := json.Marshal(history)
	return fmt.Sprintf(`Tu es un expert climat.
Voici une série temporelle d'émissions mensuelles (tCO2e) au format JSON :
%s

Objectifs :
- Fournir une prévision simple des 12 prochains mois.
- Expliquer brièvement les tendances (hausse, baisse, saisonnalité).
- Proposer 2 à 3 leviers clés de réduction à partir de cette trajectoire.

Réponds sous forme d'analyse rédigée en français, structurée en sections.`, string(data))
}

func buildReportPrompt(summary map[string]interface{}) (string, error) {
	data, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`Tu es un assistant climat qui rédige des rapports de Bilan Carbone pour des entreprises.

Voici un résumé JSON des résultats (émissions totales, par scope, par poste, évolution, etc.) :
%s

Rédige un rapport exécutif en français, clair et concis, contenant :
- Un résumé exécutif (moins de 10 lignes).
- Une analyse par scope (1, 2, 3) avec les principaux postes émetteurs.
- 5 recommandations actionnables avec estimation qualitative du gain CO2 (faible/moyen/fort).

Adopte un ton professionnel accessible à des dirigeants non experts.`, string(data))

	return prompt, nil
}


