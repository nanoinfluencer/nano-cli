package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const StateDirEnv = "NANOINF_STATE_DIR"

type Channel struct {
	ID        string                   `json:"id"`
	Platform  string                   `json:"platform"`
	Name      string                   `json:"name,omitempty"`
	Username  string                   `json:"username,omitempty"`
	URL       string                   `json:"url,omitempty"`
	Icon      string                   `json:"icon,omitempty"`
	Email     []map[string]interface{} `json:"email,omitempty"`
	Flag      string                   `json:"flag,omitempty"`
	ProjectID string                   `json:"project_id,omitempty"`
	Raw       interface{}              `json:"raw,omitempty"`
	UpdatedAt int64                    `json:"updated_at"`
}

type LastSearch struct {
	Kind      string `json:"kind,omitempty"`
	InputURL  string `json:"input_url,omitempty"`
	Platform  string `json:"platform,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	NextToken string `json:"next_token,omitempty"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}

type State struct {
	Version      int                `json:"version"`
	LastInputURL string             `json:"last_input_url,omitempty"`
	LastTaskID   string             `json:"last_task_id,omitempty"`
	LastSearch   *LastSearch        `json:"last_search,omitempty"`
	Channels     map[string]Channel `json:"channels"`
}

func Default() State {
	return State{
		Version:  1,
		Channels: map[string]Channel{},
	}
}

func ResolveDir() (string, error) {
	if dir := os.Getenv(StateDirEnv); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "nanoinf", "workspaces", "default"), nil
}

func ResolvePath() (string, error) {
	dir, err := ResolveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func Load() (State, error) {
	path, err := ResolvePath()
	if err != nil {
		return State{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return State{}, err
	}

	st := Default()
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, err
	}
	if st.Channels == nil {
		st.Channels = map[string]Channel{}
	}
	if st.Version == 0 {
		st.Version = 1
	}
	return st, nil
}

func Save(st State) error {
	path, err := ResolvePath()
	if err != nil {
		return err
	}
	if st.Version == 0 {
		st.Version = 1
	}
	if st.Channels == nil {
		st.Channels = map[string]Channel{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func ChannelKey(platform, id string) string {
	return platform + ":" + id
}

func UpsertChannel(st *State, inputURL string, ch Channel) {
	if st.Channels == nil {
		st.Channels = map[string]Channel{}
	}
	ch.UpdatedAt = time.Now().Unix()
	st.LastInputURL = inputURL
	st.Channels[ChannelKey(ch.Platform, ch.ID)] = ch
}
