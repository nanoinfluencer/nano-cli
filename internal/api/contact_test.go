package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nanoinfluencer/nano-cli/internal/config"
)

func TestGetContactAcceptsStringEmails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/contact/ytb/UC123" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"email":["hello@example.com"]}}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{BaseURL: server.URL, Token: "token"}, server.Client(), ClientMeta{})
	emails, err := client.GetContact(context.Background(), "ytb", "UC123")
	if err != nil {
		t.Fatalf("GetContact failed: %v", err)
	}
	if len(emails) != 1 || emails[0]["value"] != "hello@example.com" || emails[0]["type"] != "MATCHED" {
		t.Fatalf("unexpected emails: %#v", emails)
	}
}

func TestGetContactIncludesAPIErrorDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"contact quota exceeded","code":"CONTACT_QUOTA_EXCEEDED"}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{BaseURL: server.URL, Token: "token"}, server.Client(), ClientMeta{})
	_, err := client.GetContact(context.Background(), "ytb", "UC123")
	if err == nil {
		t.Fatalf("expected GetContact to fail")
	}
	got := err.Error()
	for _, want := range []string{
		"contact request failed with status 403 Forbidden",
		`{"error":"contact quota exceeded","code":"CONTACT_QUOTA_EXCEEDED"}`,
		"retry after 60",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q, got %s", want, got)
		}
	}
}

func TestGetContactIncludesStringDataError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"data":"Invalid Token","needAuth":true}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{BaseURL: server.URL, Token: "token"}, server.Client(), ClientMeta{})
	_, err := client.GetContact(context.Background(), "ytb", "UC123")
	if err == nil {
		t.Fatalf("expected GetContact to fail")
	}
	got := err.Error()
	if !strings.Contains(got, `contact request failed with status 403 Forbidden: {"data":"Invalid Token","needAuth":true}`) {
		t.Fatalf("unexpected error: %s", got)
	}
}
