package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"hostvra/agent/internal/collector"
	"hostvra/agent/internal/osadapter"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type EnrollRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	Name            string `json:"name"`
	Hostname        string `json:"hostname"`
	IPAddress       string `json:"ip_address"`
	OSName          string `json:"os_name"`
	OSVersion       string `json:"os_version"`
	Architecture    string `json:"architecture"`
	KernelVersion   string `json:"kernel_version"`
	AgentVersion    string `json:"agent_version"`
	CPUCores        int    `json:"cpu_cores"`
	CPUModel        string `json:"cpu_model"`
	RAMTotalMB      int64  `json:"ram_total_mb"`
	DiskTotalGB     int64  `json:"disk_total_gb"`
}

type EnrollResponseData struct {
	ServerID string `json:"server_id"`
	AgentKey string `json:"agent_key"`
	OrgID    string `json:"org_id"`
	Message  string `json:"message"`
}

type Envelope struct {
	Success bool                `json:"success"`
	Data    *EnrollResponseData `json:"data,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *APIClient) Enroll(token string, osInfo osadapter.OSAdapter, metrics *collector.MetricsSnapshot, agentVersion string) (*EnrollResponseData, error) {
	reqBody := EnrollRequest{
		EnrollmentToken: token,
		Name:            metrics.Hostname,
		Hostname:        metrics.Hostname,
		IPAddress:       metrics.IPAddress,
		OSName:          osInfo.Name(),
		OSVersion:       osInfo.Version(),
		Architecture:    osInfo.Architecture(),
		KernelVersion:   osInfo.KernelVersion(),
		AgentVersion:    agentVersion,
		CPUCores:        metrics.CPUCores,
		CPUModel:        metrics.CPUModel,
		RAMTotalMB:      metrics.RAMTotalMB,
		DiskTotalGB:     metrics.DiskTotalGB,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/agent/enroll", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("enrollment HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var env Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !env.Success || env.Data == nil {
		if env.Error != nil {
			return nil, fmt.Errorf("enrollment rejected by control plane: [%s] %s", env.Error.Code, env.Error.Message)
		}
		return nil, errors.New("enrollment rejected by control plane with unknown error")
	}

	return env.Data, nil
}

type HeartbeatPayload struct {
	UptimeSeconds int64                      `json:"uptime_seconds"`
	Metric        *collector.MetricsSnapshot `json:"metric"`
}

func (c *APIClient) SendHeartbeat(serverID, agentKey string, metrics *collector.MetricsSnapshot) error {
	reqBody := HeartbeatPayload{
		UptimeSeconds: metrics.UptimeSec,
		Metric:        metrics,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/agent/heartbeat", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", agentKey))
	req.Header.Set("X-Server-ID", serverID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("control plane returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
