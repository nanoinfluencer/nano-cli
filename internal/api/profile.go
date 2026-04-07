package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

type ResolveProfileResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Platform string `json:"platform"`
}

type ProfileResponse struct {
	Data map[string]interface{} `json:"data"`
}

func (c *Client) ResolveURL(ctx context.Context, inputURL string) (ResolveProfileResponse, error) {
	payload, err := json.Marshal(map[string]string{"url": inputURL})
	if err != nil {
		return ResolveProfileResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/profile", bytes.NewReader(payload))
	if err != nil {
		return ResolveProfileResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ResolveProfileResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return ResolveProfileResponse{}, fmt.Errorf("resolve request failed with status %d", resp.StatusCode)
	}

	var data ResolveProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ResolveProfileResponse{}, err
	}
	if data.ID == "" || data.Platform == "" {
		return ResolveProfileResponse{}, fmt.Errorf("resolve response missing id or platform")
	}
	return data, nil
}

func (c *Client) GetProfile(ctx context.Context, platform, id string) (map[string]interface{}, error) {
	if c.token == "" {
		return nil, config.ErrTokenNotConfigured
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/profile/%s/%s", c.baseURL, platform, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("profile request failed with status %d", resp.StatusCode)
	}

	var data ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Data == nil {
		return nil, fmt.Errorf("profile response missing data")
	}
	return data.Data, nil
}
