package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type FlagResponse struct {
	Data struct {
		Channels []map[string]interface{} `json:"channels"`
		Error    string                   `json:"error"`
	} `json:"data"`
	Error string `json:"error"`
}

func (c *Client) SaveFlag(ctx context.Context, payload map[string]interface{}) (FlagResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return FlagResponse{}, err
	}

	req, err := c.newRequest(ctx, "POST", c.baseURL+"/api/flag", bytes.NewReader(body), true)
	if err != nil {
		return FlagResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FlagResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return FlagResponse{}, fmt.Errorf("flag request failed with status %d", resp.StatusCode)
	}

	var data FlagResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return FlagResponse{}, err
	}
	if data.Error != "" {
		return FlagResponse{}, fmt.Errorf(data.Error)
	}
	if data.Data.Error != "" {
		return FlagResponse{}, fmt.Errorf(data.Data.Error)
	}
	return data, nil
}
