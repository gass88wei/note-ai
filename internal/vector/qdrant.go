package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// QdrantClient communicates with Qdrant via REST API.
type QdrantClient struct {
	baseURL    string
	httpClient *http.Client
	cmd        *exec.Cmd
	running    bool
}

func NewQdrantClient(baseURL string) *QdrantClient {
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &QdrantClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ===== Process Management =====

// StartQdrant tries to find and start qdrant.exe if not already running.
func (c *QdrantClient) StartQdrant() error {
	// Check if already running
	if c.IsRunning() {
		c.running = true
		return nil
	}

	// Find qdrant.exe
	qdrantPath := c.findQdrantBinary()
	if qdrantPath == "" {
		return fmt.Errorf("qdrant.exe not found. Place it in libs/ or project root")
	}

	c.cmd = exec.Command(qdrantPath)
	c.cmd.Dir = filepath.Dir(qdrantPath)
	c.cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start qdrant: %v", err)
	}

	c.running = true

	// Wait for Qdrant to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		if c.IsRunning() {
			fmt.Println("[Qdrant] Started successfully")
			return nil
		}
	}

	return fmt.Errorf("qdrant started but not responding after 15s")
}

func (c *QdrantClient) StopQdrant() {
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
		c.running = false
		fmt.Println("[Qdrant] Stopped")
	}
}

func (c *QdrantClient) IsRunning() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/healthz")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *QdrantClient) findQdrantBinary() string {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	wd, _ := os.Getwd()

	paths := []string{
		filepath.Join(wd, "qdrant.exe"),                 // CWD (project root during dev)
		filepath.Join(wd, "libs", "qdrant.exe"),         // CWD/libs
		filepath.Join(exeDir, "qdrant.exe"),             // next to exe
		filepath.Join(exeDir, "libs", "qdrant.exe"),     // exe/libs
		filepath.Join(exeDir, "..", "..", "qdrant.exe"), // build/bin -> project root
		filepath.Join(exeDir, "..", "..", "libs", "qdrant.exe"),
	}
	for _, p := range paths {
		abs, _ := filepath.Abs(p)
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	return ""
}

func (c *QdrantClient) getStorageDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".note-ai", "qdrant_storage")
}

// ===== Collection Management =====

// CreateCollection creates a Qdrant collection with the given vector size.
func (c *QdrantClient) CreateCollection(name string, vectorSize int) error {
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	return c.put("/collections/"+name, body)
}

// DeleteCollection removes a collection.
func (c *QdrantClient) DeleteCollection(name string) error {
	return c.delete("/collections/" + name)
}

// CollectionExists checks if a collection exists.
func (c *QdrantClient) CollectionExists(name string) bool {
	resp, err := c.httpClient.Get(c.baseURL + "/collections/" + name)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// GetCollectionDimension returns the vector dimension of a collection.
func (c *QdrantClient) GetCollectionDimension(name string) (int, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/collections/" + name)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	return result.Result.Config.Params.Vectors.Size, nil
}

// RecreateCollection deletes and creates a collection.
func (c *QdrantClient) RecreateCollection(name string, vectorSize int) error {
	c.DeleteCollection(name)
	return c.CreateCollection(name, vectorSize)
}

// ===== Point Operations =====

// Point represents a vector point to upsert.
type Point struct {
	ID      int64
	Vector  []float64
	Payload map[string]interface{}
}

// UpsertPoints inserts or updates points in a collection.
func (c *QdrantClient) UpsertPoints(collection string, points []Point) error {
	if len(points) == 0 {
		return nil
	}

	qdrantPoints := make([]map[string]interface{}, len(points))
	for i, p := range points {
		qdrantPoints[i] = map[string]interface{}{
			"id":     p.ID,
			"vector": p.Vector,
		}
		if len(p.Payload) > 0 {
			qdrantPoints[i]["payload"] = p.Payload
		}
	}

	body := map[string]interface{}{
		"points": qdrantPoints,
	}
	return c.put("/collections/"+collection+"/points", body)
}

// DeletePoints removes points by their IDs.
func (c *QdrantClient) DeletePoints(collection string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	body := map[string]interface{}{
		"points": ids,
	}
	return c.post("/collections/"+collection+"/points/delete", body)
}

// SearchPoints searches for nearest vectors.
func (c *QdrantClient) SearchPoints(collection string, vector []float64, topK int) ([]int64, []float64, error) {
	body := map[string]interface{}{
		"vector":       vector,
		"top":          topK,
		"with_payload": false,
		"with_vector":  false,
	}

	jsonData, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(
		c.baseURL+"/collections/"+collection+"/points/search",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("search failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("search error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result []struct {
			ID    int64   `json:"id"`
			Score float64 `json:"score"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse search result: %v", err)
	}

	ids := make([]int64, len(result.Result))
	scores := make([]float64, len(result.Result))
	for i, r := range result.Result {
		ids[i] = r.ID
		scores[i] = r.Score
	}
	return ids, scores, nil
}

// ScrollPoints returns all point IDs in a collection (for reindexing).
func (c *QdrantClient) ScrollPoints(collection string, limit int) ([]int64, string, error) {
	url := fmt.Sprintf("%s/collections/%s/points/scroll?limit=%d&with_payload=false&with_vector=false",
		c.baseURL, collection, limit)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result struct {
			Points         []map[string]interface{} `json:"points"`
			NextPageOffset *int64                   `json:"next_page_offset"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", err
	}

	ids := make([]int64, len(result.Result.Points))
	for i, p := range result.Result.Points {
		if id, ok := p["id"].(float64); ok {
			ids[i] = int64(id)
		}
	}

	offset := ""
	if result.Result.NextPageOffset != nil {
		offset = fmt.Sprintf("%d", *result.Result.NextPageOffset)
	}
	return ids, offset, nil
}

// CountPoints returns the number of points in a collection.
func (c *QdrantClient) CountPoints(collection string) (int, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/collections/" + collection)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result struct {
			PointsCount int `json:"points_count"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	return result.Result.PointsCount, nil
}

// ===== HTTP Helpers =====

func (c *QdrantClient) put(path string, body interface{}) error {
	jsonData, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", c.baseURL+path, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("PUT %s returned %d", path, resp.StatusCode)
	}
	return nil
}

func (c *QdrantClient) post(path string, body interface{}) error {
	jsonData, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s returned %d", path, resp.StatusCode)
	}
	return nil
}

func (c *QdrantClient) delete(path string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+path, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
