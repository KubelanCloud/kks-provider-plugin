package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KubelanCloud/kks-csi-plugin/pkg/csi/provisioner"
)

type ClientConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		token:   strings.TrimSpace(cfg.Token),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) ClusterInfo(ctx context.Context) (provisioner.ClusterInfo, error) {
	var out provisioner.ClusterInfo
	if err := c.doJSON(ctx, http.MethodGet, "/v1/cluster", nil, &out); err != nil {
		return provisioner.ClusterInfo{}, err
	}
	return out, nil
}

func (c *Client) CreateVolume(ctx context.Context, req provisioner.CreateVolumeRequest) (provisioner.Volume, error) {
	var out provisioner.Volume
	if err := c.doJSON(ctx, http.MethodPost, "/v1/volumes", req, &out); err != nil {
		return provisioner.Volume{}, err
	}
	return out, nil
}

func (c *Client) DeleteVolume(ctx context.Context, volumeID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/v1/volumes/"+escapePath(volumeID), nil, nil)
}

func (c *Client) VolumeExists(ctx context.Context, volumeID string) (bool, error) {
	var out provisioner.VolumeExistsResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/volumes/"+escapePath(volumeID), nil, &out)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return out.Exists, nil
}

func (c *Client) PublishVolume(ctx context.Context, volumeID, nodeID string) (provisioner.PublishVolumeResponse, error) {
	var out provisioner.PublishVolumeResponse
	req := provisioner.PublishVolumeRequest{NodeID: nodeID}
	path := "/v1/volumes/" + escapePath(volumeID) + "/publish"
	if err := c.doJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return provisioner.PublishVolumeResponse{}, err
	}
	return out, nil
}

func (c *Client) UnpublishVolume(ctx context.Context, volumeID, nodeID string) error {
	req := provisioner.UnpublishVolumeRequest{
		NodeID:   nodeID,
		VolumeID: volumeID,
	}
	return c.doJSON(ctx, http.MethodPost, "/v1/volumes/unpublish", req, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		raw, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       path,
			Body:       strings.TrimSpace(string(raw)),
		}
	}

	if respBody == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, respBody); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

type HTTPError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s %s: status %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("%s %s: status %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}

func isNotFound(err error) bool {
	httpErr, ok := err.(*HTTPError)
	return ok && httpErr.StatusCode == http.StatusNotFound
}

func escapePath(value string) string {
	return strings.ReplaceAll(value, "/", "%2F")
}
