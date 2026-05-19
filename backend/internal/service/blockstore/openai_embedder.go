package blockstoreservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIEmbedder struct {
	apiKey  string
	model   string
	dims    int
	baseURL string
	hc      *http.Client
}

type OpenAIEmbedderConfig struct {
	APIKey  string
	Model   string
	Dims    int
	BaseURL string
	Timeout time.Duration
}

func NewOpenAIEmbedder(cfg OpenAIEmbedderConfig) (*OpenAIEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("openai embedder: APIKey is required")
	}
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.Dims == 0 {
		cfg.Dims = 1536
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &OpenAIEmbedder{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		dims:    cfg.Dims,
		baseURL: cfg.BaseURL,
		hc:      &http.Client{Timeout: cfg.Timeout},
	}, nil
}

func (e *OpenAIEmbedder) Model() string { return e.model }
func (e *OpenAIEmbedder) Dims() int     { return e.dims }

type openaiEmbedRequest struct {
	Input      string `json:"input"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type openaiEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := openaiEmbedRequest{Input: text, Model: e.model}
	// 3-small/3-large support `dimensions` reduction — pass it so stored vectors match DB column width.
	if e.dims != 0 {
		reqBody.Dimensions = e.dims
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out openaiEmbedResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("openai embed: decode: %w (body=%s)", err, string(raw))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("openai embed: %s (%s)", out.Error.Message, out.Error.Type)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai embed: HTTP %d: %s", resp.StatusCode, string(raw))
	}
	if len(out.Data) == 0 {
		return nil, errors.New("openai embed: empty data")
	}
	vec := out.Data[0].Embedding
	if len(vec) != e.dims {
		e.dims = len(vec)
	}
	return vec, nil
}
