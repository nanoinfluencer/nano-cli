package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	meta       ClientMeta
}

type ClientMeta struct {
	Version  string
	AppID    string
	Platform string
	DeviceID string
}

type WhoAmIResponse struct {
	OK   bool `json:"ok,omitempty"`
	User struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"user"`
	CLI struct {
		Enabled  bool   `json:"enabled"`
		Scope    string `json:"scope"`
		Client   string `json:"client"`
		Version  string `json:"version"`
		AppID    string `json:"appId"`
		Platform string `json:"platform"`
		DeviceID string `json:"deviceId"`
	} `json:"cli,omitempty"`
}

func NewClient(cfg config.Config, httpClient *http.Client, meta ClientMeta) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		token:      cfg.Token,
		httpClient: httpClient,
		meta:       meta,
	}
}

func (c *Client) newRequest(ctx context.Context, method string, url string, body io.Reader, withAuth bool) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if withAuth {
		if c.token == "" {
			return nil, config.ErrTokenNotConfigured
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("X-NAINF-CLIENT", "cli")
	req.Header.Set("app-version", c.meta.Version)
	req.Header.Set("app-id", c.meta.AppID)
	req.Header.Set("app-platform", c.meta.Platform)
	req.Header.Set("app-device-id", c.meta.DeviceID)
	req.Header.Set("User-Agent", fmt.Sprintf("nanoinf/%s", c.meta.Version))
	return req, nil
}

func (c *Client) WhoAmI(ctx context.Context) (WhoAmIResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.baseURL+"/api/cli/whoami", nil, true)
	if err != nil {
		return WhoAmIResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return WhoAmIResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
			if apiErr.Code != "" {
				return WhoAmIResponse{}, fmt.Errorf("%s (%s)", apiErr.Error, apiErr.Code)
			}
			return WhoAmIResponse{}, fmt.Errorf("%s", apiErr.Error)
		}
		return WhoAmIResponse{}, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var data WhoAmIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return WhoAmIResponse{}, err
	}
	return data, nil
}
