package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

func TestSearchSimilarRetriesMissingJobID(t *testing.T) {
	oldDelay := searchRetryDelay
	searchRetryDelay = 0
	t.Cleanup(func() {
		searchRetryDelay = oldDelay
	})

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/search/ytb" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts < 3 {
			_, _ = w.Write([]byte(`{"message":"temporary search startup miss","data":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"job_id":"job-123"},"message":"Success"}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{BaseURL: server.URL, Token: "token"}, server.Client(), ClientMeta{})
	resp, err := client.SearchSimilar(context.Background(), "ytb", "UC123", nil, SearchTags{}, nil, nil)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}
	if resp.Data.JobID != "job-123" {
		t.Fatalf("unexpected job id: %s", resp.Data.JobID)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestSearchSimilarMissingJobIDIncludesResponseDetails(t *testing.T) {
	oldDelay := searchRetryDelay
	searchRetryDelay = 0
	t.Cleanup(func() {
		searchRetryDelay = oldDelay
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"search busy","code":"SEARCH_BUSY","data":{"retry":true}}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{BaseURL: server.URL, Token: "token"}, server.Client(), ClientMeta{})
	_, err := client.SearchSimilar(context.Background(), "ytb", "UC123", nil, SearchTags{}, nil, nil)
	if err == nil {
		t.Fatalf("expected SearchSimilar to fail")
	}
	got := err.Error()
	for _, want := range []string{
		"search response missing job_id",
		"message=search busy",
		"code=SEARCH_BUSY",
		`data={"retry":true}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q, got %s", want, got)
		}
	}
}

func TestSleepWithContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(ctx, time.Second); err == nil {
		t.Fatalf("expected canceled context error")
	}
}
