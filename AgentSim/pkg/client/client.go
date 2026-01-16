package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/types"
)

// Client provides interface to AgentSim control API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new AgentSim client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetStatus retrieves current simulation status
func (c *Client) GetStatus() (*types.SimulationStatus, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/status", c.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var status types.SimulationStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Start starts the simulation with the specified number of active agents
func (c *Client) Start(activeAgents int) error {
	req := map[string]int{"activeAgents": activeAgents}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/start", c.baseURL),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start simulation: %s", string(body))
	}

	return nil
}

// Stop stops the simulation
func (c *Client) Stop() error {
	resp, err := c.httpClient.Post(fmt.Sprintf("%s/stop", c.baseURL), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop simulation: %s", string(body))
	}

	return nil
}

// GetConfig retrieves current configuration
func (c *Client) GetConfig() (*types.SimulationConfig, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/config", c.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config types.SimulationConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Health checks if the service is healthy
func (c *Client) Health() error {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/health", c.baseURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy: status code %d", resp.StatusCode)
	}

	return nil
}
