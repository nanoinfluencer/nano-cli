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

type SearchTags struct {
	PosTags []string
	NegTags []string
}

var searchRetryDelay = time.Second

func (c *Client) SearchSimilar(ctx context.Context, platform, channelID string, filters map[string]interface{}, tags SearchTags, cursor *SearchCursor, excludeCIDs []string) (SearchResponse, error) {
	payload := map[string]interface{}{
		"cid": channelID,
	}
	if len(filters) > 0 {
		payload["filters"] = filters
	}
	if len(tags.PosTags) > 0 {
		payload["posTags"] = tags.PosTags
	}
	if len(tags.NegTags) > 0 {
		payload["negTags"] = tags.NegTags
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

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		data, err := c.searchSimilarOnce(ctx, platform, body)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !isMissingJobIDError(err) || attempt == 3 {
			break
		}
		if err := sleepWithContext(ctx, searchRetryDelay*time.Duration(attempt)); err != nil {
			return SearchResponse{}, err
		}
	}
	return SearchResponse{}, lastErr
}

func (c *Client) searchSimilarOnce(ctx context.Context, platform string, body []byte) (SearchResponse, error) {
	req, err := c.newRequest(ctx, "POST", fmt.Sprintf("%s/api/search/%s", c.baseURL, platform), bytes.NewReader(body), true)
	if err != nil {
		return SearchResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return SearchResponse{}, responseError("search request", resp)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return SearchResponse{}, err
	}
	var data SearchResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		return SearchResponse{}, err
	}
	if data.Data.JobID == "" {
		return SearchResponse{}, missingJobIDError(raw)
	}
	return data, nil
}

func missingJobIDError(raw []byte) error {
	detail := strings.TrimSpace(string(raw))
	var envelope struct {
		Message string      `json:"message"`
		Error   interface{} `json:"error"`
		Code    interface{} `json:"code"`
		Data    interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		parts := make([]string, 0, 4)
		if envelope.Message != "" {
			parts = append(parts, "message="+envelope.Message)
		}
		if envelope.Error != nil {
			parts = append(parts, fmt.Sprintf("error=%v", envelope.Error))
		}
		if envelope.Code != nil {
			parts = append(parts, fmt.Sprintf("code=%v", envelope.Code))
		}
		if envelope.Data != nil {
			encoded, err := json.Marshal(envelope.Data)
			if err == nil {
				parts = append(parts, "data="+string(encoded))
			}
		}
		if len(parts) > 0 {
			detail = strings.Join(parts, "; ")
		}
	}
	if detail == "" {
		return missingJobIDResponseError("search response missing job_id")
	}
	return missingJobIDResponseError("search response missing job_id: " + detail)
}

type missingJobIDResponseError string

func (e missingJobIDResponseError) Error() string {
	return string(e)
}

func isMissingJobIDError(err error) bool {
	_, ok := err.(missingJobIDResponseError)
	return ok
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) taskBaseURL() string {
	if strings.Contains(c.baseURL, "localhost") || strings.Contains(c.baseURL, "127.0.0.1") {
		return c.baseURL
	}
	return "https://api.nanoinfluencer.ai"
}

func (c *Client) GetTask(ctx context.Context, taskID string) (TaskResponse, error) {
	req, err := c.newRequest(ctx, "POST", fmt.Sprintf("%s/nano-api/search/task/%s", c.taskBaseURL(), taskID), bytes.NewReader([]byte(`{"channelIds":[]}`)), false)
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
		return TaskResponse{}, responseError("task request", resp)
	}

	var data TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return TaskResponse{}, err
	}
	return data, nil
}
