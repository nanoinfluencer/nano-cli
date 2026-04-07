package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

type SearchResponse struct {
	Data struct {
		JobID string `json:"job_id"`
	} `json:"data"`
	Message string `json:"message"`
}

type TaskResponse struct {
	Data struct {
		Data struct {
			Channels  []map[string]interface{} `json:"channels"`
			NextToken string                   `json:"nextToken"`
			NextIDs   interface{}              `json:"nextIds"`
		} `json:"data"`
		Meta struct {
			Channels []map[string]interface{} `json:"channels"`
			NextIDs  interface{}              `json:"nextIds"`
			Filtered []map[string]interface{} `json:"filtered"`
			Progress float64                  `json:"progress"`
			Action   string                   `json:"action"`
		} `json:"meta"`
		Status string `json:"status"`
		Job    string `json:"job"`
	} `json:"data"`
	Message    string `json:"message"`
	Pos        int    `json:"pos"`
	NextReqInt int    `json:"nextReqInt"`
}

type SearchCursor struct {
	NextToken string
	NextIDs   interface{}
}

func (c *Client) SearchSimilar(ctx context.Context, platform, channelID string, filters map[string]interface{}, cursor *SearchCursor, excludeCIDs []string) (SearchResponse, error) {
	if c.token == "" {
		return SearchResponse{}, config.ErrTokenNotConfigured
	}

	payload := map[string]interface{}{
		"cid": channelID,
	}
	if len(filters) > 0 {
		payload["filters"] = filters
	}
	if cursor != nil {
		if cursor.NextToken != "" {
			payload["nextToken"] = cursor.NextToken
		}
		if cursor.NextIDs != nil {
			payload["nextIds"] = cursor.NextIDs
		}
		payload["more"] = true
		if len(excludeCIDs) > 0 {
			payload["exclude_cids"] = excludeCIDs
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return SearchResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/search/%s", c.baseURL, platform), bytes.NewReader(body))
	if err != nil {
		return SearchResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return SearchResponse{}, fmt.Errorf("search request failed with status %d", resp.StatusCode)
	}

	var data SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return SearchResponse{}, err
	}
	if data.Data.JobID == "" {
		return SearchResponse{}, fmt.Errorf("search response missing job_id")
	}
	return data, nil
}

func (c *Client) taskBaseURL() string {
	if strings.Contains(c.baseURL, "localhost") || strings.Contains(c.baseURL, "127.0.0.1") {
		return c.baseURL
	}
	return "https://api.nanoinfluencer.ai"
}

func (c *Client) GetTask(ctx context.Context, taskID string) (TaskResponse, error) {
	if c.token == "" {
		return TaskResponse{}, config.ErrTokenNotConfigured
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/nano-api/search/task/%s", c.taskBaseURL(), taskID), bytes.NewReader([]byte(`{"channelIds":[]}`)))
	if err != nil {
		return TaskResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TaskResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return TaskResponse{}, fmt.Errorf("task request failed with status %d", resp.StatusCode)
	}

	var data TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return TaskResponse{}, err
	}
	return data, nil
}
