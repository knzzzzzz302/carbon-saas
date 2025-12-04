package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
			Timeout: 20 * time.Second,
		},
	}
}

type mistralInvocationRequest struct {
	// Le champ "input" est utilisé par l'API Agents de Mistral pour fournir le texte.
	Input string `json:"input"`
}

type mistralInvocationResponse struct {
	Output string `json:"output"`
}

func (m *MistralClient) enabled() bool {
	return m != nil && m.apiKey != "" && m.agentID != ""
}

func (m *MistralClient) invokeAgent(prompt string) (string, error) {
	if !m.enabled() {
		return "", errors.New("mistral non configuré")
	}

	body, err := json.Marshal(mistralInvocationRequest{Input: prompt})
	if err != nil {
		return "", err
	}

	url := mistralBaseURL + "/agents/" + m.agentID + "/invocations"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", errors.New("appel Mistral échoué avec le statut " + resp.Status)
	}

	var parsed mistralInvocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}

	return parsed.Output, nil
}


