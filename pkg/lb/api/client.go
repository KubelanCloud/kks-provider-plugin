package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KubelanCloud/kks-provider-plugin/pkg/lb/provisioner"
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

func (c *Client) Allocate(ctx context.Context, req provisioner.AllocateRequest) (provisioner.LoadBalancer, error) {
	var out provisioner.LoadBalancer
	if err := c.doJSON(ctx, http.MethodPost, "/v1/loadbalancers", req, &out); err != nil {
		return provisioner.LoadBalancer{}, err
	}
	return out, nil
}

func (c *Client) Get(ctx context.Context, id string) (provisioner.LoadBalancer, error) {
	var out provisioner.LoadBalancer
	if err := c.doJSON(ctx, http.MethodGet, "/v1/loadbalancers/"+escapePath(id), nil, &out); err != nil {
		return provisioner.LoadBalancer{}, err
	}
	return out, nil
}

func (c *Client) Release(ctx context.Context, id string) error {
	err := c.doJSON(ctx, http.MethodDelete, "/v1/loadbalancers/"+escapePath(id), nil, nil)
	if isNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		message := strings.TrimSpace(string(raw))
		var payload struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &payload) == nil && payload.Error != "" {
			message = payload.Error
		}
		if message == "" {
			message = resp.Status
		}
		if resp.StatusCode == http.StatusNotFound {
			return NewHTTPError(http.StatusNotFound, message)
		}
		return fmt.Errorf("lb api %s %s: %s", method, path, message)
	}
	if out == nil || len(raw) == 0 || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode lb api response: %w", err)
	}
	return nil
}

func escapePath(value string) string {
	return strings.ReplaceAll(value, "/", "%2F")
}

func isNotFound(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound
}
