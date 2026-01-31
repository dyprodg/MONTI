package callgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CallAPIClient sends enqueue requests to the backend call API.
type CallAPIClient struct {
	backendURL string
	httpClient *http.Client
}

// NewCallAPIClient creates a new client pointing at the given backend base URL
// (e.g. "http://localhost:8080").
func NewCallAPIClient(backendURL string) *CallAPIClient {
	return &CallAPIClient{
		backendURL: backendURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// enqueueRequest is the JSON body sent to the backend.
type enqueueRequest struct {
	VQ     string `json:"vq"`
	CallID string `json:"callId"`
}

// EnqueueCall posts a new call to /internal/call/enqueue with a generated UUID.
func (c *CallAPIClient) EnqueueCall(vqName string) error {
	callID := uuid.New().String()

	body, err := json.Marshal(enqueueRequest{
		VQ:     vqName,
		CallID: callID,
	})
	if err != nil {
		return fmt.Errorf("marshal enqueue request: %w", err)
	}

	url := c.backendURL + "/internal/call/enqueue"
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s returned status %d", url, resp.StatusCode)
	}

	return nil
}
