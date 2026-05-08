package api

import (
	"context"
	"net/http"
	"net/http/httptest"
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
