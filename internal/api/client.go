package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type WhoAmIResponse struct {
	User struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Image string `json:"image"`
	} `json:"user"`
	Token string `json:"token,omitempty"`
}

func NewClient(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		token:      cfg.Token,
		httpClient: httpClient,
	}
}

func (c *Client) WhoAmI(ctx context.Context) (WhoAmIResponse, error) {
	if c.token == "" {
		return WhoAmIResponse{}, config.ErrTokenNotConfigured
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/user", nil)
	if err != nil {
		return WhoAmIResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return WhoAmIResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return WhoAmIResponse{}, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var data WhoAmIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return WhoAmIResponse{}, err
	}
	return data, nil
}
