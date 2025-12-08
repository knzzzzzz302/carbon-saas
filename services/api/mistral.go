package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// Client léger pour appeler Mistral via un Agent.

const mistralBaseURL = "https://api.mistral.ai/v1"

type MistralClient struct {
	apiKey  string
	agentID string
	client  *http.Client
}

func NewMistralClient(cfg Config) *MistralClient {
	return &MistralClient{
		apiKey:  cfg.MistralAPIKey,
		agentID: cfg.MistralAgentID,
		client: &http.Client{
			// Certaines réponses peuvent être un peu longues sur des questions techniques,
			// on laisse donc une marge confortable avant de considérer que c'est un timeout.
			Timeout: 60 * time.Second,
		},
	}
}

// Pour coller au curl officiel Mistral que tu utilises :
//   POST https://api.mistral.ai/v1/conversations
//   Headers: X-API-KEY: <clé>
//   Body: { "agent_id": "...", "inputs": "Hello there!" }
type mistralInvocationRequest struct {
	AgentID string `json:"agent_id"`
	Inputs  string `json:"inputs"`
}

// La réponse complète des conversations est riche, mais pour le chatbot
// on peut simplement renvoyer le corps brut, lisible dans le frontend.

func (m *MistralClient) enabled() bool {
	return m != nil && m.apiKey != "" && m.agentID != ""
}

func (m *MistralClient) invokeAgent(prompt string) (string, error) {
	if !m.enabled() {
		return "", errors.New("mistral non configuré")
	}

	body, err := json.Marshal(mistralInvocationRequest{
		AgentID: m.agentID,
		Inputs:  prompt,
	})
	if err != nil {
		return "", err
	}

	// On aligne le client Go sur le curl Mistral fourni par l'utilisateur.
	url := mistralBaseURL + "/conversations"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	// Le curl officiel utilise X-API-KEY ; on fait la même chose.
	req.Header.Set("X-API-KEY", m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", errors.New("appel Mistral échoué avec le statut " + resp.Status)
	}

	// Pour simplifier et rester robuste face aux changements de schéma,
	// on renvoie le corps brut sous forme de string ; le frontend l'affiche tel quel.
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if len(b) == 0 {
		return "", errors.New("réponse Mistral vide")
	}
	return string(b), nil
}


