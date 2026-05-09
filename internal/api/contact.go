package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ContactResponse struct {
	Data struct {
		Email interface{} `json:"email"`
	} `json:"data"`
	Error string `json:"error"`
}

func (c *Client) GetContact(ctx context.Context, platform, id string) ([]map[string]interface{}, error) {
	req, err := c.newRequest(ctx, "GET", fmt.Sprintf("%s/api/contact/%s/%s", c.baseURL, platform, id), nil, true)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, responseError("contact request", resp)
	}

	var data ContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error != "" {
		return nil, fmt.Errorf(data.Error)
	}
	return normalizeContactEmails(data.Data.Email), nil
}

func normalizeContactEmails(v interface{}) []map[string]interface{} {
	list, ok := v.([]interface{})
	if !ok {
		if typed, ok := v.([]map[string]interface{}); ok {
			return typed
		}
		return nil
	}

	out := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		switch value := item.(type) {
		case map[string]interface{}:
			out = append(out, value)
		case string:
			out = append(out, map[string]interface{}{"type": "MATCHED", "value": value})
		}
	}
	return out
}
