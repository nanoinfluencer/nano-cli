package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

type ContactResponse struct {
	Data struct {
		Email []map[string]interface{} `json:"email"`
	} `json:"data"`
	Error string `json:"error"`
}

func (c *Client) GetContact(ctx context.Context, platform, id string) ([]map[string]interface{}, error) {
	if c.token == "" {
		return nil, config.ErrTokenNotConfigured
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/contact/%s/%s", c.baseURL, platform, id), nil)
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
		return nil, fmt.Errorf("contact request failed with status %d", resp.StatusCode)
	}

	var data ContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error != "" {
		return nil, fmt.Errorf(data.Error)
	}
	return data.Data.Email, nil
}
