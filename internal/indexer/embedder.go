package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embedder calls LM Studio / OpenAI compatible embedding API.
type Embedder struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
	dimension  int
}

func NewEmbedder(baseURL, model, apiKey string) *Embedder {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Embedder{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EmbedSingle returns the embedding vector for a single text.
func (e *Embedder) EmbedSingle(text string) ([]float64, error) {
	vecs, err := e.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return vecs[0], nil
}

// EmbedBatch returns embedding vectors for multiple texts.
func (e *Embedder) EmbedBatch(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := map[string]interface{}{
		"model": e.model,
		"input": texts,
	}
	jsonData, _ := json.Marshal(reqBody)

	apiURL := e.baseURL + "/embeddings"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API call failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embedding API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse embedding response: %v", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("embedding API returned no data")
	}

	vecs := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		if d.Index < len(vecs) {
			vecs[d.Index] = d.Embedding
		}
	}

	if e.dimension == 0 && len(vecs[0]) > 0 {
		e.dimension = len(vecs[0])
	}
	return vecs, nil
}

// Dimension returns the detected embedding dimension.
func (e *Embedder) Dimension() int { return e.dimension }

// TestConnection checks if the embedding API is reachable.
func (e *Embedder) TestConnection() (bool, string) {
	_, err := e.EmbedSingle("test")
	if err != nil {
		return false, fmt.Sprintf("Embedding API error: %v", err)
	}
	return true, fmt.Sprintf("Embedding API OK (dimension: %d)", e.dimension)
}
