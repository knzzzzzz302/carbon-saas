package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultBaseURL     = "https://api.mistral.ai"
	defaultEmbedModel  = "mistral-embed"
	conversationPath   = "/v1/conversations"
	embeddingPath      = "/v1/embeddings"
	defaultHTTPTimeout = 30 * time.Second
)

type Client struct {
	apiKey  string
	agentID string
	baseURL string
	http    *http.Client
}

type ConversationRequest struct {
	AgentID string      `json:"agent_id"`
	Inputs  interface{} `json:"inputs"`
}

type ConversationResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Status  string               `json:"status"`
	Message ConversationPiece    `json:"message"`
	Outputs []ConversationOutput `json:"outputs"`
	Output  any                  `json:"output"`
}

type ConversationPiece struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ConversationOutput struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Role    string              `json:"role"`
	Content []ConversationChunk `json:"content"`
}

type ConversationChunk struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type EmbeddingRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		return nil, errors.New("MISTRAL_API_KEY manquant")
	}
	agentID := os.Getenv("MISTRAL_AGENT_ID")
	baseURL := os.Getenv("MISTRAL_API_BASE")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		apiKey:  apiKey,
		agentID: agentID,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}, nil
}

func (c *Client) SendConversation(ctx context.Context, prompt string) (*ConversationResponse, error) {
	if c.agentID == "" {
		return nil, errors.New("MISTRAL_AGENT_ID manquant")
	}
	payload := ConversationRequest{
		AgentID: c.agentID,
		Inputs:  prompt,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+conversationPath, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mistral conversation status %d", resp.StatusCode)
	}

	var out ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	model := defaultEmbedModel
	payload := EmbeddingRequest{
		Model: model,
		Input: text,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+embeddingPath, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mistral embed status %d", resp.StatusCode)
	}

	var out EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, errors.New("embedding vide")
	}
	return out.Data[0].Embedding, nil
}

func (r *ConversationResponse) FirstText() string {
	if r == nil {
		return ""
	}
	if r.Message.Content != "" {
		return r.Message.Content
	}
	for _, out := range r.Outputs {
		for _, chunk := range out.Content {
			if chunk.Text != "" {
				return chunk.Text
			}
		}
	}
	if text, ok := r.Output.(string); ok && text != "" {
		return text
	}
	return ""
}
