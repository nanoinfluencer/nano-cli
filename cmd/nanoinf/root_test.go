package nanoinf

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nanoinfluencer/nano-cli/internal/api"
	"github.com/nanoinfluencer/nano-cli/internal/config"
	"github.com/nanoinfluencer/nano-cli/internal/state"
)

const testToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6Imxlby56aGFvanVuQGdtYWlsLmNvbSIsImNoZWNrc3VtIjoiajhla20iLCJpYXQiOjE3NzU1MjkwODcsImV4cCI6MTc4MDcxMzA4N30.QVy4ak9NqaJrYV_MZ7E8ICDNRDnywz_KN8NywwGF6Kg"

func TestAuthTokenSetWritesConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/whoami" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		assertCLIHeaders(t, r)
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"user":{"email":"leo.zhaojun@gmail.com","name":"Leo","image":""},"cli":{"enabled":true,"scope":"cli","client":"cli","version":"dev","appId":"nanoinf-cli","platform":"` + runtime.GOOS + `","deviceId":"device-123"}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{
		BaseURL: server.URL,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "auth", "token", "set", testToken)
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}

	if !strings.Contains(stdout, `"ok": true`) {
		t.Fatalf("expected success output, got %s", stdout)
	}

	cfgPath := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if cfg.Token != testToken {
		t.Fatalf("expected token to be saved")
	}
	if cfg.BaseURL != server.URL {
		t.Fatalf("expected saved base url, got %s", cfg.BaseURL)
	}
	if cfg.DeviceID == "" {
		t.Fatalf("expected device id to be saved")
	}
}

func TestAuthStatusWithoutToken(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)

	stdout, stderr, err := execute(t, Dependencies{}, "auth", "status")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}

	if !strings.Contains(stdout, `"configured": false`) {
		t.Fatalf("expected configured false, got %s", stdout)
	}
}

func TestConfigShowRedactsToken(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)

	if err := config.Save(config.Config{
		BaseURL: config.DefaultBaseURL,
		Token:   testToken,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{}, "config", "show")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}

	if !strings.Contains(stdout, `"has_token": true`) {
		t.Fatalf("expected has_token true, got %s", stdout)
	}
	if strings.Contains(stdout, testToken) {
		t.Fatalf("expected token to be redacted in config show output")
	}
	if !strings.Contains(stdout, config.PreviewToken(testToken)) {
		t.Fatalf("expected token preview in output, got %s", stdout)
	}
}

func TestWhoAmIUsesBearerToken(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/whoami" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		assertCLIHeaders(t, r)
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"user":{"email":"leo.zhaojun@gmail.com","name":"Leo","image":""},"cli":{"enabled":true,"scope":"cli","client":"cli","version":"dev","appId":"nanoinf-cli","platform":"` + runtime.GOOS + `","deviceId":"device-123"}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{
		BaseURL: server.URL,
		Token:   testToken,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "whoami")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}

	if !strings.Contains(stdout, `"email": "leo.zhaojun@gmail.com"`) {
		t.Fatalf("expected email in output, got %s", stdout)
	}
	if !strings.Contains(stdout, `"has_token": true`) {
		t.Fatalf("expected has_token true, got %s", stdout)
	}
}

func TestWhoAmIFailsWithoutToken(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)

	_, stderr, err := execute(t, Dependencies{}, "whoami")
	if err == nil {
		t.Fatalf("expected whoami to fail without token")
	}
	if !strings.Contains(err.Error(), "access token not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %s", stderr)
	}
}

func assertCLIHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("X-NAINF-CLIENT"); got != "cli" {
		t.Fatalf("unexpected cli header: %s", got)
	}
	if got := r.Header.Get("app-version"); got == "" {
		t.Fatalf("missing app-version header")
	}
	if got := r.Header.Get("app-id"); got != "nanoinf-cli" {
		t.Fatalf("unexpected app-id header: %s", got)
	}
	if got := r.Header.Get("app-platform"); got != runtime.GOOS {
		t.Fatalf("unexpected app-platform header: %s", got)
	}
	if got := r.Header.Get("app-device-id"); got == "" {
		t.Fatalf("missing app-device-id header")
	}
	if got := r.Header.Get("User-Agent"); !strings.Contains(got, "nanoinf/") {
		t.Fatalf("unexpected user-agent header: %s", got)
	}
}

func TestRootCommandResolvesURLAndWritesState(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/profile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"UC123","name":"The AI Search","icon":"https://img.example/icon.png","platform":"ytb"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/profile/ytb/UC123":
			if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
				t.Fatalf("unexpected authorization header: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"UC123","platform":"ytb","name":"The AI Search","username":"theAIsearch","url":"https://www.youtube.com/@theAIsearch","icon":"https://img.example/icon.png","email":[{"type":"email","value":"hello@example.com"}],"flag":"","project_id":"0"}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := config.Save(config.Config{
		BaseURL: server.URL,
		Token:   testToken,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "https://www.youtube.com/@theAIsearch")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"platform": "ytb"`) {
		t.Fatalf("expected platform in output, got %s", stdout)
	}
	if !strings.Contains(stdout, `"id": "UC123"`) {
		t.Fatalf("expected id in output, got %s", stdout)
	}

	statePath := filepath.Join(stateDir, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}

	var st state.State
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	ch, ok := st.Channels["ytb:UC123"]
	if !ok {
		t.Fatalf("expected channel to be saved in state")
	}
	if ch.Name != "The AI Search" {
		t.Fatalf("unexpected channel name: %s", ch.Name)
	}
	if st.LastInputURL != "https://www.youtube.com/@theAIsearch" {
		t.Fatalf("unexpected last input url: %s", st.LastInputURL)
	}
}

func TestRootCommandFailsWithoutTokenAfterResolve(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/profile" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"UC123","name":"The AI Search","icon":"https://img.example/icon.png","platform":"ytb"}`))
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	if err := config.Save(config.Config{BaseURL: server.URL}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	_, _, err := execute(t, Dependencies{HTTPClient: server.Client()}, "https://www.youtube.com/@theAIsearch")
	if err == nil {
		t.Fatalf("expected command to fail without token")
	}
	if !strings.Contains(err.Error(), "access token not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSimilarCommandWritesChannelsAndReturnsNextToken(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/profile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"UCIgnGlGkVRhd4qNFcEwLL4A","name":"AI Search","icon":"https://img.example/icon.png","platform":"ytb"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/ytb":
			if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
				t.Fatalf("unexpected authorization header: %s", got)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			filters, ok := body["filters"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected filters payload, got %#v", body["filters"])
			}
			if filters["email"] != true {
				t.Fatalf("expected has-email filter, got %#v", filters["email"])
			}
			if filters["lastPostDays"] != float64(30) {
				t.Fatalf("expected active-within filter, got %#v", filters["lastPostDays"])
			}
			assertFloatArray(t, filters["country"], []float64{840, 826})
			assertFloatArray(t, filters["excludeCountry"], []float64{392})
			assertFloatArray(t, filters["subs"], []float64{10000, 200000})
			assertFloatArray(t, filters["views"], []float64{1000, 50000})
			assertFloatArray(t, filters["posts"], []float64{10, 500})
			assertFloatArray(t, filters["er"], []float64{2, 20})
			assertFloatArray(t, filters["vr"], []float64{5, 50})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"job_id":"job-123"},"message":"Success"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/nano-api/search/task/job-123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"job":"job-123","status":"finished","meta":{"channels":[{"id":"UCaaa","platform":"ytb","name":"AI Research","username":"airesearch","url":"https://youtube.com/channel/UCaaa","icon":"https://img.example/a.jpg","email":[{"type":"MATCHED","value":"hello@example.com"}]},{"id":"UCbbb","platform":"ytb","name":"Gradient Update","username":"gradientupdate","url":"https://youtube.com/channel/UCbbb","icon":"https://img.example/b.jpg","email":[{"type":"REVEAL BUTTON","value":""}]}]},"data":{"channels":[{"id":"UCaaa","platform":"ytb","name":"AI Research","username":"airesearch","url":"https://youtube.com/channel/UCaaa","icon":"https://img.example/a.jpg","email":[{"type":"MATCHED","value":"hello@example.com"}]},{"id":"UCbbb","platform":"ytb","name":"Gradient Update","username":"gradientupdate","url":"https://youtube.com/channel/UCbbb","icon":"https://img.example/b.jpg","email":[{"type":"REVEAL BUTTON","value":""}]}],"nextToken":"ytb:cursor-token","nextIds":["cursor-1","cursor-2"]}},"message":"Success"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := config.Save(config.Config{
		BaseURL: server.URL,
		Token:   testToken,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(
		t,
		Dependencies{HTTPClient: server.Client()},
		"similar",
		"https://www.youtube.com/@theAIsearch",
		"--has-email",
		"--country", "US",
		"--country", "GB",
		"--exclude-country", "JP",
		"--active-within", "30",
		"--subs", "10000:200000",
		"--views", "1000:50000",
		"--posts", "10:500",
		"--er", "2:20",
		"--vr", "5:50",
	)
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"task_id": "job-123"`) {
		t.Fatalf("expected task id in output, got %s", stdout)
	}
	if !strings.Contains(stdout, `"id": "UCaaa"`) || !strings.Contains(stdout, `"id": "UCbbb"`) {
		t.Fatalf("expected channels in output, got %s", stdout)
	}
	expectedNext, err := encodeNextToken(api.SearchCursor{
		NextToken: "ytb:cursor-token",
		NextIDs:   []string{"cursor-1", "cursor-2"},
	})
	if err != nil {
		t.Fatalf("encode next token: %v", err)
	}
	if !strings.Contains(stdout, expectedNext) {
		t.Fatalf("expected next token in output, got %s", stdout)
	}

	st, err := state.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if st.LastTaskID != "job-123" {
		t.Fatalf("unexpected last task id: %s", st.LastTaskID)
	}
	if st.LastSearch == nil {
		t.Fatalf("expected last search to be saved")
	}
	if st.LastSearch.InputURL != "https://www.youtube.com/@theAIsearch" {
		t.Fatalf("unexpected last search input url: %s", st.LastSearch.InputURL)
	}
	if st.LastSearch.NextToken != expectedNext {
		t.Fatalf("unexpected saved next token: %s", st.LastSearch.NextToken)
	}
	if st.LastSearch.Filters == nil {
		t.Fatalf("expected last search filters to be saved")
	}
	assertFloatArray(t, st.LastSearch.Filters["country"], []float64{840, 826})
	assertFloatArray(t, st.LastSearch.Filters["excludeCountry"], []float64{392})
	if _, ok := st.Channels["ytb:UCaaa"]; !ok {
		t.Fatalf("expected first channel in state")
	}
	if _, ok := st.Channels["ytb:UCbbb"]; !ok {
		t.Fatalf("expected second channel in state")
	}
}

func TestSimilarCommandPassesNextTokenToSearch(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	nextToken, err := encodeNextToken(api.SearchCursor{
		NextToken: "ytb:cursor-token",
		NextIDs:   []string{"cursor-1", "cursor-2"},
	})
	if err != nil {
		t.Fatalf("encode next token: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/profile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"UCIgnGlGkVRhd4qNFcEwLL4A","name":"AI Search","icon":"https://img.example/icon.png","platform":"ytb"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/ytb":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got, _ := body["nextToken"].(string); got != "ytb:cursor-token" {
				t.Fatalf("unexpected nextToken payload: %#v", body["nextToken"])
			}
			got, ok := body["nextIds"].([]interface{})
			if !ok || len(got) != 2 || got[0] != "cursor-1" || got[1] != "cursor-2" {
				t.Fatalf("unexpected nextIds payload: %#v", body["nextIds"])
			}
			if more, _ := body["more"].(bool); !more {
				t.Fatalf("expected more=true payload: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"job_id":"job-124"},"message":"Success"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/nano-api/search/task/job-124":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"job":"job-124","status":"finished","meta":{"channels":[]},"data":{"channels":[]}},"message":"Success"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := config.Save(config.Config{
		BaseURL: server.URL,
		Token:   testToken,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	_, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "similar", "https://www.youtube.com/@theAIsearch", "--next", nextToken)
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
}

func TestNextCommandUsesSavedLastSearch(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	nextToken, err := encodeNextToken(api.SearchCursor{
		NextToken: "ytb:cursor-token",
		NextIDs:   []string{"cursor-1", "cursor-2"},
	})
	if err != nil {
		t.Fatalf("encode next token: %v", err)
	}

	st := state.Default()
	st.LastSearch = &state.LastSearch{
		Kind:      "similar",
		InputURL:  "https://www.youtube.com/@theAIsearch",
		Platform:  "ytb",
		ChannelID: "UCIgnGlGkVRhd4qNFcEwLL4A",
		NextToken: nextToken,
		Filters: map[string]interface{}{
			"email": true,
			"country": []interface{}{
				float64(840),
			},
			"subs": []interface{}{
				float64(10000),
				float64(200000),
			},
		},
	}
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/profile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"UCIgnGlGkVRhd4qNFcEwLL4A","name":"AI Search","icon":"https://img.example/icon.png","platform":"ytb"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/ytb":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got, _ := body["nextToken"].(string); got != "ytb:cursor-token" {
				t.Fatalf("unexpected nextToken payload: %#v", body["nextToken"])
			}
			filters, ok := body["filters"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected filters payload, got %#v", body["filters"])
			}
			if filters["email"] != true {
				t.Fatalf("expected saved email filter, got %#v", filters["email"])
			}
			assertFloatArray(t, filters["country"], []float64{840})
			assertFloatArray(t, filters["subs"], []float64{10000, 200000})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"job_id":"job-next"},"message":"Success"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/nano-api/search/task/job-next":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"job":"job-next","status":"finished","meta":{"channels":[]},"data":{"channels":[{"id":"UCnext","platform":"ytb","name":"Next Result","username":"nextresult","url":"https://youtube.com/channel/UCnext","icon":"https://img.example/next.jpg","email":[{"type":"MATCHED","value":"next@example.com"}]}]}},"message":"Success"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := config.Save(config.Config{
		BaseURL: server.URL,
		Token:   testToken,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "next")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"id": "UCnext"`) {
		t.Fatalf("expected next-page channel in output, got %s", stdout)
	}
}

func TestNextCommandFailsWithoutSavedSearch(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	_, _, err := execute(t, Dependencies{}, "next")
	if err == nil {
		t.Fatalf("expected next to fail without saved search")
	}
	if !strings.Contains(err.Error(), "no recent similar search") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactGetUsesWorkspaceWhenContactExists(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	st := state.Default()
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC123",
		Platform: "ytb",
		Name:     "AI Search",
		Email: []map[string]interface{}{
			{"type": "MATCHED", "value": "hello@example.com"},
		},
	})
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{}, "contact", "get", "--platform", "ytb", "--id", "UC123")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"source": "workspace"`) {
		t.Fatalf("expected workspace source, got %s", stdout)
	}
	if !strings.Contains(stdout, `hello@example.com`) {
		t.Fatalf("expected local email in output, got %s", stdout)
	}
}

func TestContactGetFetchesAndMergesWhenWorkspaceHasNoValidContact(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	st := state.Default()
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC123",
		Platform: "ytb",
		Name:     "AI Search",
		Email: []map[string]interface{}{
			{"type": "REVEAL BUTTON", "value": ""},
		},
		Raw: map[string]interface{}{
			"id": "UC123",
		},
	})
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/contact/ytb/UC123" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"email":[{"type":"MATCHED","value":"hello@example.com"}]}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{BaseURL: server.URL, Token: testToken}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "contact", "get", "--platform", "ytb", "--id", "UC123")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"source": "api"`) {
		t.Fatalf("expected api source, got %s", stdout)
	}
	if !strings.Contains(stdout, `hello@example.com`) {
		t.Fatalf("expected fetched email in output, got %s", stdout)
	}

	updated, err := state.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	ch := updated.Channels["ytb:UC123"]
	if len(ch.Email) < 2 {
		t.Fatalf("expected merged contact in state, got %#v", ch.Email)
	}
}

func TestContactFillEnrichesOnlyChannelsWithoutContact(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	st := state.Default()
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC1",
		Platform: "ytb",
		Name:     "Need Contact",
		Email: []map[string]interface{}{
			{"type": "REVEAL BUTTON", "value": ""},
		},
		Raw: map[string]interface{}{"id": "UC1"},
	})
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC2",
		Platform: "ytb",
		Name:     "Has Contact",
		Email: []map[string]interface{}{
			{"type": "MATCHED", "value": "exists@example.com"},
		},
		Raw: map[string]interface{}{"id": "UC2"},
	})
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/contact/ytb/UC1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"email":[{"type":"MATCHED","value":"filled@example.com"}]}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{BaseURL: server.URL, Token: testToken}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "contact", "fill", "--limit", "10")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if calls != 1 {
		t.Fatalf("expected exactly one contact API call, got %d", calls)
	}
	if !strings.Contains(stdout, `"updated": 1`) {
		t.Fatalf("expected one updated channel, got %s", stdout)
	}

	updated, err := state.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if !hasUsableContact(updated.Channels["ytb:UC1"].Email) {
		t.Fatalf("expected UC1 to have usable contact")
	}
	if updated.Channels["ytb:UC2"].Email[0]["value"] != "exists@example.com" {
		t.Fatalf("expected UC2 contact to stay unchanged")
	}
}

func TestFavoriteAddRequiresLocalChannel(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	_, _, err := execute(t, Dependencies{}, "favorite", "add", "--platform", "ytb", "--id", "UC404")
	if err == nil {
		t.Fatalf("expected favorite add to fail without local channel")
	}
	if !strings.Contains(err.Error(), "channel not found in local workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFavoriteAddPostsFullChannelPayloadAndUpdatesState(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	st := state.Default()
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:        "UC123",
		Platform:  "ytb",
		Name:      "AI Search",
		Username:  "theaisearch",
		URL:       "https://www.youtube.com/@theAIsearch",
		Icon:      "https://img.example/icon.png",
		ProjectID: "12",
		Email: []map[string]interface{}{
			{"type": "MATCHED", "value": "hello@example.com"},
		},
	})
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/flag" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["group_id"] != "12" {
			t.Fatalf("expected group_id 12, got %#v", body["group_id"])
		}
		channels, ok := body["channels"].([]interface{})
		if !ok || len(channels) != 1 {
			t.Fatalf("unexpected channels payload: %#v", body["channels"])
		}
		channel := channels[0].(map[string]interface{})
		if channel["id"] != "UC123" || channel["platform"] != "ytb" || channel["flag"] != "fav" {
			t.Fatalf("unexpected channel payload: %#v", channel)
		}
		if channel["name"] != "AI Search" || channel["url"] != "https://www.youtube.com/@theAIsearch" {
			t.Fatalf("expected full channel payload, got %#v", channel)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"channels":[{"id":"UC123","platform":"ytb","flag":"fav"}]}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{BaseURL: server.URL, Token: testToken}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "favorite", "add", "--platform", "ytb", "--id", "UC123")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"flag": "fav"`) {
		t.Fatalf("expected fav flag in output, got %s", stdout)
	}

	updated, err := state.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if updated.Channels["ytb:UC123"].Flag != "fav" {
		t.Fatalf("expected state flag to be fav")
	}
}

func TestFavoriteFillFavoritesOnlyEligibleChannels(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	st := state.Default()
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC1",
		Platform: "ytb",
		Name:     "Eligible",
		Email: []map[string]interface{}{
			{"type": "MATCHED", "value": "eligible@example.com"},
		},
	})
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC2",
		Platform: "ytb",
		Name:     "No Contact",
		Email: []map[string]interface{}{
			{"type": "REVEAL BUTTON", "value": ""},
		},
	})
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:        "UC3",
		Platform:  "ytb",
		Name:      "Already Favorited",
		Flag:      "fav",
		ProjectID: "12",
		Email: []map[string]interface{}{
			{"type": "MATCHED", "value": "already@example.com"},
		},
	})
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/flag" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		calls++
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		channels := body["channels"].([]interface{})
		channel := channels[0].(map[string]interface{})
		if channel["id"] != "UC1" {
			t.Fatalf("expected only UC1 to be favorited, got %#v", channel)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"channels":[{"id":"UC1","platform":"ytb","flag":"fav"}]}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{BaseURL: server.URL, Token: testToken}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	stdout, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "favorite", "fill", "--project", "12", "--limit", "10")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}
	if calls != 1 {
		t.Fatalf("expected one favorite API call, got %d", calls)
	}
	if !strings.Contains(stdout, `"favorited": 1`) {
		t.Fatalf("expected one favorited channel, got %s", stdout)
	}

	updated, err := state.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if updated.Channels["ytb:UC1"].Flag != "fav" || updated.Channels["ytb:UC1"].ProjectID != "12" {
		t.Fatalf("expected UC1 to be favorited in state")
	}
	if updated.Channels["ytb:UC2"].Flag != "" {
		t.Fatalf("expected UC2 to stay untouched")
	}
}

func TestHideAddAllowsExplicitProjectOverride(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv(state.StateDirEnv, stateDir)

	st := state.Default()
	state.UpsertChannel(&st, "https://www.youtube.com/@theAIsearch", state.Channel{
		ID:       "UC123",
		Platform: "ytb",
		Name:     "AI Search",
	})
	if err := state.Save(st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["group_id"] != "34" {
			t.Fatalf("expected override group_id 34, got %#v", body["group_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"channels":[{"id":"UC123","platform":"ytb","flag":"hide"}]}}`))
	}))
	defer server.Close()

	if err := config.Save(config.Config{BaseURL: server.URL, Token: testToken}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	_, stderr, err := execute(t, Dependencies{HTTPClient: server.Client()}, "hide", "add", "--platform", "ytb", "--id", "UC123", "--project", "34")
	if err != nil {
		t.Fatalf("execute failed: %v, stderr=%s", err, stderr)
	}

	updated, err := state.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	ch := updated.Channels["ytb:UC123"]
	if ch.Flag != "hide" || ch.ProjectID != "34" {
		t.Fatalf("unexpected updated state: %#v", ch)
	}
}

func execute(t *testing.T, deps Dependencies, args ...string) (string, string, error) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := NewRootCommandWithDeps(deps)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func assertFloatArray(t *testing.T, value interface{}, expected []float64) {
	t.Helper()

	got, ok := value.([]interface{})
	if !ok {
		t.Fatalf("expected array payload, got %#v", value)
	}
	if len(got) != len(expected) {
		t.Fatalf("unexpected array length: got %#v want %#v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("unexpected array value at %d: got %#v want %#v", i, got, expected)
		}
	}
}
