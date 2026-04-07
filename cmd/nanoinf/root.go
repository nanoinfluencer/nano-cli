package nanoinf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nanoinfluencer/nano-cli/internal/api"
	"github.com/nanoinfluencer/nano-cli/internal/config"
	"github.com/nanoinfluencer/nano-cli/internal/country"
	"github.com/nanoinfluencer/nano-cli/internal/state"
)

type Dependencies struct {
	HTTPClient *http.Client
}

func NewRootCommand() *cobra.Command {
	return NewRootCommandWithDeps(Dependencies{})
}

func NewRootCommandWithDeps(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "nanoinf",
		Short:         "NanoInfluencer CLI for seed-based creator discovery",
		Long:          "NanoInfluencer CLI helps agents and operators resolve creator URLs, find similar influencers, enrich contact data, and save channels into favorites or hide lists using local workspace state.",
		Example:       "  nanoinf https://www.youtube.com/@theAIsearch\n  nanoinf similar https://www.youtube.com/@theAIsearch\n  nanoinf contact get --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A\n  nanoinf favorite add --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A --project 12",
		Version:       "dev",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runResolveURL(cmd, deps, args[0])
		},
	}

	cmd.AddCommand(newAuthCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newWhoAmICommand(deps))
	cmd.AddCommand(newSimilarCommand(deps))
	cmd.AddCommand(newNextCommand(deps))
	cmd.AddCommand(newContactCommand(deps))
	cmd.AddCommand(newFavoriteCommand(deps))
	cmd.AddCommand(newHideCommand(deps))

	return cmd
}

func writeJSON(cmd *cobra.Command, v interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  "Manage the access token used by nanoinf to call NanoInfluencer APIs.",
	}
	authCmd.AddCommand(newAuthTokenCommand())
	authCmd.AddCommand(newAuthStatusCommand())
	return authCmd
}

func newAuthTokenCommand() *cobra.Command {
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Manage access token",
		Long:  "Save and inspect the local access token used for CLI requests.",
	}

	tokenCmd.AddCommand(&cobra.Command{
		Use:     "set <token>",
		Short:   "Save access token",
		Example: `  nanoinf auth token set <token>`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Token = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}

			path, err := config.ResolvePath()
			if err != nil {
				return err
			}
			return writeJSON(cmd, map[string]interface{}{
				"ok":            true,
				"config_path":   path,
				"token_preview": config.PreviewToken(cfg.Token),
			})
		},
	})

	return tokenCmd
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show auth status",
		Example: `  nanoinf auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			path, err := config.ResolvePath()
			if err != nil {
				return err
			}

			return writeJSON(cmd, map[string]interface{}{
				"configured":    cfg.Token != "",
				"base_url":      cfg.BaseURL,
				"config_path":   path,
				"token_preview": config.PreviewToken(cfg.Token),
			})
		},
	}
}

func newConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage config",
		Long:  "Inspect the local nanoinf configuration file and resolved base URL.",
	}

	configCmd.AddCommand(&cobra.Command{
		Use:     "show",
		Short:   "Show config",
		Example: `  nanoinf config show`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			path, err := config.ResolvePath()
			if err != nil {
				return err
			}

			return writeJSON(cmd, map[string]interface{}{
				"base_url":      cfg.BaseURL,
				"config_path":   path,
				"has_token":     cfg.Token != "",
				"token_preview": config.PreviewToken(cfg.Token),
			})
		},
	})

	return configCmd
}

func newWhoAmICommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "whoami",
		Short:   "Show current user",
		Long:    "Call the NanoInfluencer user endpoint with the configured bearer token and show the current user profile.",
		Example: `  nanoinf whoami`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			client := api.NewClient(cfg, deps.HTTPClient)
			resp, err := client.WhoAmI(cmd.Context())
			if err != nil {
				if err == config.ErrTokenNotConfigured {
					return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
				}
				return err
			}

			return writeJSON(cmd, map[string]interface{}{
				"user":      resp.User,
				"has_token": resp.Token != "",
				"base_url":  cfg.BaseURL,
			})
		},
	}
}

func newSimilarCommand(deps Dependencies) *cobra.Command {
	var nextToken string
	var hasEmail bool
	var countries []string
	var excludeCountries []string
	var activeWithin int
	var subsRange string
	var viewsRange string
	var postsRange string
	var erRange string
	var vrRange string

	cmd := &cobra.Command{
		Use:     "similar <url>",
		Short:   "Find similar influencers from a seed URL",
		Long:    "Resolve a creator URL, run a similar-influencer search, store the returned channels in local workspace state, and return an optional next token for pagination.",
		Example: "  nanoinf similar https://www.youtube.com/@theAIsearch\n  nanoinf similar https://www.youtube.com/@theAIsearch --has-email --country US --country GB --active-within 30 --subs 10000:200000\n  nanoinf similar https://www.youtube.com/@theAIsearch --next <token>",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filters, err := buildSearchFilters(hasEmail, countries, excludeCountries, activeWithin, subsRange, viewsRange, postsRange, erRange, vrRange)
			if err != nil {
				return err
			}
			return runSimilar(cmd, deps, similarOptions{
				InputURL:  args[0],
				NextToken: nextToken,
				Filters:   filters,
			})
		},
	}

	cmd.Flags().StringVar(&nextToken, "next", "", "Opaque pagination token")
	cmd.Flags().BoolVar(&hasEmail, "has-email", false, "Only return channels with usable email")
	cmd.Flags().StringArrayVar(&countries, "country", nil, "Include country by alpha-2 code, repeatable (example: --country US --country GB)")
	cmd.Flags().StringArrayVar(&excludeCountries, "exclude-country", nil, "Exclude country by alpha-2 code, repeatable")
	cmd.Flags().IntVar(&activeWithin, "active-within", 0, "Only return channels published within the last N days")
	cmd.Flags().StringVar(&subsRange, "subs", "", "Subscriber range as min:max")
	cmd.Flags().StringVar(&viewsRange, "views", "", "Median views range as min:max")
	cmd.Flags().StringVar(&postsRange, "posts", "", "Post count range as min:max")
	cmd.Flags().StringVar(&erRange, "er", "", "Engagement rate percentage range as min:max")
	cmd.Flags().StringVar(&vrRange, "vr", "", "View rate percentage range as min:max")
	return cmd
}

func newNextCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "next",
		Short:   "Fetch the next page from the last similar search",
		Long:    "Continue the most recent similar search stored in the local workspace by reusing its saved pagination token.",
		Example: "  nanoinf next",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load()
			if err != nil {
				return err
			}
			if st.LastSearch == nil || st.LastSearch.Kind != "similar" || st.LastSearch.InputURL == "" {
				return fmt.Errorf("no recent similar search found in local workspace")
			}
			if st.LastSearch.NextToken == "" {
				return fmt.Errorf("no next page available for the last similar search")
			}
			return runSimilar(cmd, deps, similarOptions{
				InputURL:  st.LastSearch.InputURL,
				NextToken: st.LastSearch.NextToken,
				Filters:   st.LastSearch.Filters,
			})
		},
	}
}

func Execute() error {
	cmd := NewRootCommand()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	return cmd.Execute()
}

func runResolveURL(cmd *cobra.Command, deps Dependencies, inputURL string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg, deps.HTTPClient)
	resolved, err := client.ResolveURL(cmd.Context(), inputURL)
	if err != nil {
		return err
	}

	profile, err := client.GetProfile(cmd.Context(), resolved.Platform, resolved.ID)
	if err != nil {
		if err == config.ErrTokenNotConfigured {
			return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
		}
		return err
	}

	channel := state.Channel{
		ID:       stringValue(profile["id"], resolved.ID),
		Platform: stringValue(profile["platform"], resolved.Platform),
		Name:     stringValue(profile["name"], resolved.Name),
		Username: stringValue(profile["username"], ""),
		URL:      stringValue(profile["url"], inputURL),
		Icon:     stringValue(profile["icon"], resolved.Icon),
		Flag:     stringValue(profile["flag"], ""),
		Raw:      profile,
		Email:    emailSlice(profile["email"]),
	}

	st, err := state.Load()
	if err != nil {
		return err
	}
	state.UpsertChannel(&st, inputURL, channel)
	if err := state.Save(st); err != nil {
		return err
	}

	statePath, err := state.ResolvePath()
	if err != nil {
		return err
	}

	return writeJSON(cmd, map[string]interface{}{
		"channel":    channel,
		"input_url":  inputURL,
		"state_path": statePath,
	})
}

func stringValue(v interface{}, fallback string) string {
	s, ok := v.(string)
	if ok && s != "" {
		return s
	}
	return fallback
}

type similarOptions struct {
	InputURL  string
	NextToken string
	Filters   map[string]interface{}
}

func runSimilar(cmd *cobra.Command, deps Dependencies, opts similarOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg, deps.HTTPClient)
	resolved, err := client.ResolveURL(cmd.Context(), opts.InputURL)
	if err != nil {
		return err
	}

	cursor, err := decodeNextToken(opts.NextToken)
	if err != nil {
		return err
	}

	st, err := state.Load()
	if err != nil {
		return err
	}

	searchResp, err := client.SearchSimilar(cmd.Context(), resolved.Platform, resolved.ID, opts.Filters, cursor, excludeCIDs(st, resolved.Platform))
	if err != nil {
		if err == config.ErrTokenNotConfigured {
			return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
		}
		return err
	}

	taskResp, err := pollTask(cmd.Context(), client, searchResp.Data.JobID)
	if err != nil {
		return err
	}

	rawChannels := taskResp.Data.Data.Channels
	if len(rawChannels) == 0 {
		rawChannels = taskResp.Data.Meta.Channels
	}

	channels := make([]state.Channel, 0, len(rawChannels))
	for _, item := range rawChannels {
		ch := state.Channel{
			ID:        stringValue(item["id"], ""),
			Platform:  stringValue(item["platform"], resolved.Platform),
			Name:      stringValue(item["name"], ""),
			Username:  stringValue(item["username"], ""),
			URL:       stringValue(item["url"], ""),
			Icon:      stringValue(item["icon"], ""),
			Flag:      stringValue(item["flag"], ""),
			ProjectID: stringValue(item["project_id"], ""),
			Raw:       item,
			Email:     emailSlice(item["email"]),
		}
		if ch.ID == "" {
			continue
		}
		state.UpsertChannel(&st, opts.InputURL, ch)
		channels = append(channels, ch)
	}

	var encodedNext string
	if taskResp.Data.Data.NextToken != "" || taskResp.Data.Data.NextIDs != nil {
		encodedNext, err = encodeNextToken(api.SearchCursor{
			NextToken: taskResp.Data.Data.NextToken,
			NextIDs:   taskResp.Data.Data.NextIDs,
		})
		if err != nil {
			return err
		}
	}

	st.LastTaskID = searchResp.Data.JobID
	st.LastSearch = &state.LastSearch{
		Kind:      "similar",
		InputURL:  opts.InputURL,
		Platform:  resolved.Platform,
		ChannelID: resolved.ID,
		NextToken: encodedNext,
		Filters:   cloneMap(opts.Filters),
		UpdatedAt: time.Now().Unix(),
	}
	if err := state.Save(st); err != nil {
		return err
	}

	return writeJSON(cmd, map[string]interface{}{
		"seed": map[string]string{
			"id":       resolved.ID,
			"platform": resolved.Platform,
			"url":      opts.InputURL,
		},
		"task_id":    searchResp.Data.JobID,
		"channels":   channels,
		"filters":    opts.Filters,
		"next_token": encodedNext,
	})
}

func buildSearchFilters(hasEmail bool, countries, excludeCountries []string, activeWithin int, subsRange, viewsRange, postsRange, erRange, vrRange string) (map[string]interface{}, error) {
	filters := map[string]interface{}{}

	if hasEmail {
		filters["email"] = true
	}
	if activeWithin > 0 {
		filters["lastPostDays"] = activeWithin
	}

	if len(countries) > 0 {
		codes, err := country.ResolveAlpha2List(countries)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 {
			filters["country"] = codes
		}
	}
	if len(excludeCountries) > 0 {
		codes, err := country.ResolveAlpha2List(excludeCountries)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 {
			filters["excludeCountry"] = codes
		}
	}

	rangeInputs := map[string]string{
		"subs":  subsRange,
		"views": viewsRange,
		"posts": postsRange,
		"er":    erRange,
		"vr":    vrRange,
	}
	for key, value := range rangeInputs {
		if value == "" {
			continue
		}
		parsed, err := parseRange(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --%s: %w", key, err)
		}
		filters[key] = parsed
	}

	if len(filters) == 0 {
		return nil, nil
	}
	return filters, nil
}

func parseRange(value string) ([]int, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected min:max")
	}

	out := make([]int, 0, 2)
	if parts[0] != "" {
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid min value")
		}
		out = append(out, v)
	}
	if parts[1] != "" {
		v, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid max value")
		}
		if len(out) == 0 {
			out = append(out, 0)
		}
		if len(out) == 1 && out[0] > v {
			out = []int{v, out[0]}
		} else {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("range cannot be empty")
	}
	return out, nil
}

func cloneMap(v map[string]interface{}) map[string]interface{} {
	if len(v) == 0 {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

func pollTask(ctx context.Context, client *api.Client, taskID string) (api.TaskResponse, error) {
	renderProgress := newTaskProgressRenderer()
	defer renderProgress.finish()

	deadline := time.Now().Add(100 * time.Second)
	for {
		resp, err := client.GetTask(ctx, taskID)
		if err != nil {
			return api.TaskResponse{}, err
		}
		renderProgress.update(resp)

		status := resp.Data.Status
		if status == "finished" || time.Now().After(deadline) {
			return resp, nil
		}

		select {
		case <-ctx.Done():
			return api.TaskResponse{}, ctx.Err()
		case <-time.After(taskPollDelay(resp)):
		}
	}
}

type taskProgressRenderer struct {
	enabled bool
	last    string
}

func newTaskProgressRenderer() *taskProgressRenderer {
	info, err := os.Stderr.Stat()
	if err != nil {
		return &taskProgressRenderer{}
	}
	return &taskProgressRenderer{
		enabled: (info.Mode() & os.ModeCharDevice) != 0,
	}
}

func (r *taskProgressRenderer) update(resp api.TaskResponse) {
	if !r.enabled {
		return
	}

	status := resp.Data.Status
	message := ""
	switch status {
	case "queued":
		if resp.Pos > 0 {
			message = fmt.Sprintf("Searching... queued (%d ahead)", resp.Pos)
		} else {
			message = "Searching... queued"
		}
	case "started":
		if resp.Data.Meta.Progress > 0 {
			message = fmt.Sprintf("Searching... %.1f%%", resp.Data.Meta.Progress*100)
		} else if len(resp.Data.Meta.Channels) > 0 {
			message = fmt.Sprintf("Searching... %d found", len(resp.Data.Meta.Channels))
		} else {
			message = "Searching..."
		}
	case "finished":
		message = "Searching... done"
	default:
		if status != "" {
			message = fmt.Sprintf("Searching... %s", status)
		}
	}

	if message == "" || message == r.last {
		return
	}
	r.last = message
	_, _ = fmt.Fprintf(os.Stderr, "\r%s", padProgressMessage(message))
}

func (r *taskProgressRenderer) finish() {
	if !r.enabled || r.last == "" {
		return
	}
	_, _ = fmt.Fprint(os.Stderr, "\n")
}

func padProgressMessage(message string) string {
	const width = 48
	if len(message) >= width {
		return message
	}
	return message + strings.Repeat(" ", width-len(message))
}

func taskPollDelay(resp api.TaskResponse) time.Duration {
	channels := len(resp.Data.Meta.Channels)
	if channels == 0 {
		return 3 * time.Second
	}
	secs := channels / 2
	if secs < 3 {
		secs = 3
	}
	if secs > 10 {
		secs = 10
	}
	return time.Duration(secs) * time.Second
}

func excludeCIDs(st state.State, platform string) []string {
	out := make([]string, 0, len(st.Channels))
	for _, ch := range st.Channels {
		if ch.Platform == platform && ch.ID != "" {
			out = append(out, ch.ID)
		}
	}
	return out
}

func encodeNextToken(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func decodeNextToken(token string) (*api.SearchCursor, error) {
	if token == "" {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid next token")
	}
	var cursor api.SearchCursor
	if err := json.Unmarshal(data, &cursor); err == nil && (cursor.NextToken != "" || cursor.NextIDs != nil) {
		return &cursor, nil
	}
	var legacy interface{}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("invalid next token")
	}
	return &api.SearchCursor{NextIDs: legacy}, nil
}

func newContactCommand(deps Dependencies) *cobra.Command {
	contactCmd := &cobra.Command{
		Use:   "contact",
		Short: "Get channel contact details",
		Long:  "Read contact data from the local workspace first, and only call the remote contact API when the saved channel does not already contain a usable contact.",
	}

	var platform string
	var id string
	getCmd := &cobra.Command{
		Use:     "get",
		Short:   "Get contact info for a channel",
		Example: `  nanoinf contact get --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if platform == "" || id == "" {
				return fmt.Errorf("--platform and --id are required")
			}
			return runContactGet(cmd, deps, platform, id)
		},
	}
	getCmd.Flags().StringVar(&platform, "platform", "", "Channel platform")
	getCmd.Flags().StringVar(&id, "id", "", "Channel ID")
	contactCmd.AddCommand(getCmd)

	var fillLimit int
	fillCmd := &cobra.Command{
		Use:     "fill",
		Short:   "Fetch contact info for workspace channels in batch",
		Example: "  nanoinf contact fill --limit 20",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContactFill(cmd, deps, fillLimit)
		},
	}
	fillCmd.Flags().IntVar(&fillLimit, "limit", 20, "Maximum number of channels to enrich")
	contactCmd.AddCommand(fillCmd)

	return contactCmd
}

func runContactGet(cmd *cobra.Command, deps Dependencies, platform, id string) error {
	st, err := state.Load()
	if err != nil {
		return err
	}

	key := state.ChannelKey(platform, id)
	channel, ok := st.Channels[key]
	if !ok {
		return fmt.Errorf("channel not found in local workspace")
	}

	if hasUsableContact(channel.Email) {
		return writeJSON(cmd, map[string]interface{}{
			"channel":     channel,
			"contact":     channel.Email,
			"source":      "workspace",
			"workspace":   true,
			"channel_key": key,
		})
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg, deps.HTTPClient)
	emails, err := client.GetContact(cmd.Context(), platform, id)
	if err != nil {
		if err == config.ErrTokenNotConfigured {
			return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
		}
		return err
	}

	channel.Email = mergeEmails(channel.Email, emails)
	channel.Raw = mergeRawEmail(channel.Raw, channel.Email)
	state.UpsertChannel(&st, st.LastInputURL, channel)
	if err := state.Save(st); err != nil {
		return err
	}

	return writeJSON(cmd, map[string]interface{}{
		"channel":     channel,
		"contact":     channel.Email,
		"source":      "api",
		"workspace":   true,
		"channel_key": key,
	})
}

func runContactFill(cmd *cobra.Command, deps Dependencies, limit int) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = 20
	}

	processed := 0
	updated := 0
	for _, channel := range st.Channels {
		if processed >= limit {
			break
		}
		if hasUsableContact(channel.Email) {
			continue
		}
		processed++
		if ok, err := enrichContact(cmd.Context(), deps, &st, channel); err != nil {
			if err == config.ErrTokenNotConfigured {
				return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
			}
			return err
		} else if ok {
			updated++
		}
	}

	if err := state.Save(st); err != nil {
		return err
	}

	return writeJSON(cmd, map[string]interface{}{
		"ok":        true,
		"processed": processed,
		"updated":   updated,
		"limit":     limit,
	})
}

func enrichContact(ctx context.Context, deps Dependencies, st *state.State, channel state.Channel) (bool, error) {
	if hasUsableContact(channel.Email) {
		return false, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return false, err
	}

	client := api.NewClient(cfg, deps.HTTPClient)
	emails, err := client.GetContact(ctx, channel.Platform, channel.ID)
	if err != nil {
		return false, err
	}

	merged := mergeEmails(channel.Email, emails)
	if len(merged) == len(channel.Email) {
		return false, nil
	}
	channel.Email = merged
	channel.Raw = mergeRawEmail(channel.Raw, channel.Email)
	state.UpsertChannel(st, st.LastInputURL, channel)
	return hasUsableContact(channel.Email), nil
}

func emailSlice(v interface{}) []map[string]interface{} {
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

func hasUsableContact(emails []map[string]interface{}) bool {
	for _, email := range emails {
		value, _ := email["value"].(string)
		if value != "" && !isURL(value) && containsAt(value) {
			return true
		}
	}
	return false
}

func containsAt(v string) bool {
	for _, ch := range v {
		if ch == '@' {
			return true
		}
	}
	return false
}

func isURL(v string) bool {
	return len(v) >= 8 && (v[:8] == "https://" || (len(v) >= 7 && v[:7] == "http://"))
}

func mergeEmails(existing, incoming []map[string]interface{}) []map[string]interface{} {
	seen := map[string]bool{}
	out := make([]map[string]interface{}, 0, len(existing)+len(incoming))
	for _, email := range append(existing, incoming...) {
		b, err := json.Marshal(email)
		if err != nil {
			continue
		}
		key := string(b)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, email)
	}
	return out
}

func mergeRawEmail(raw interface{}, emails []map[string]interface{}) interface{} {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return raw
	}
	m["email"] = emails
	return m
}

func newFavoriteCommand(deps Dependencies) *cobra.Command {
	return newFlagCommand(deps, "favorite", "fav")
}

func newHideCommand(deps Dependencies) *cobra.Command {
	return newFlagCommand(deps, "hide", "hide")
}

func newFlagCommand(deps Dependencies, use, flagType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: fmt.Sprintf("Manage %s channels", use),
		Long:  fmt.Sprintf("Save channels from the local workspace into the %s list. The channel must already exist in local state so nanoinf can submit the full channel payload.", use),
	}

	var platform string
	var id string
	var project string
	addCmd := &cobra.Command{
		Use:     "add",
		Short:   fmt.Sprintf("Add channel to %s", use),
		Example: fmt.Sprintf("  nanoinf %s add --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A\n  nanoinf %s add --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A --project 12", use, use),
		RunE: func(cmd *cobra.Command, args []string) error {
			if platform == "" || id == "" {
				return fmt.Errorf("--platform and --id are required")
			}
			return runFlagAdd(cmd, deps, flagType, platform, id, project)
		},
	}
	addCmd.Flags().StringVar(&platform, "platform", "", "Channel platform")
	addCmd.Flags().StringVar(&id, "id", "", "Channel ID")
	addCmd.Flags().StringVar(&project, "project", "", "Project ID")
	cmd.AddCommand(addCmd)

	if flagType == "fav" {
		var fillProject string
		var fillLimit int
		fillCmd := &cobra.Command{
			Use:     "fill",
			Short:   "Favorite workspace channels with usable email in batch",
			Example: "  nanoinf favorite fill --project 12 --limit 20",
			RunE: func(cmd *cobra.Command, args []string) error {
				if fillProject == "" {
					return fmt.Errorf("--project is required")
				}
				return runFavoriteFill(cmd, deps, fillProject, fillLimit)
			},
		}
		fillCmd.Flags().StringVar(&fillProject, "project", "", "Project ID")
		fillCmd.Flags().IntVar(&fillLimit, "limit", 20, "Maximum number of channels to favorite")
		cmd.AddCommand(fillCmd)
	}

	return cmd
}

func runFlagAdd(cmd *cobra.Command, deps Dependencies, flagType, platform, id, project string) error {
	st, err := state.Load()
	if err != nil {
		return err
	}

	key := state.ChannelKey(platform, id)
	channel, ok := st.Channels[key]
	if !ok {
		return fmt.Errorf("channel not found in local workspace")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg, deps.HTTPClient)
	flagged := map[string]interface{}{
		"id":       channel.ID,
		"platform": channel.Platform,
		"name":     channel.Name,
		"username": channel.Username,
		"icon":     channel.Icon,
		"url":      channel.URL,
		"flag":     flagType,
		"email":    channel.Email,
	}

	groupID := project
	if groupID == "" {
		groupID = channel.ProjectID
	}
	if groupID != "" {
		flagged["group_id"] = groupID
	}

	payload := map[string]interface{}{
		"channels": []map[string]interface{}{flagged},
	}
	if groupID != "" {
		payload["group_id"] = groupID
	}

	resp, err := client.SaveFlag(cmd.Context(), payload)
	if err != nil {
		if err == config.ErrTokenNotConfigured {
			return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
		}
		return err
	}

	channel.Flag = flagType
	if groupID != "" {
		channel.ProjectID = groupID
	}
	if len(resp.Data.Channels) > 0 {
		if updatedFlag := stringValue(resp.Data.Channels[0]["flag"], ""); updatedFlag != "" {
			channel.Flag = updatedFlag
		}
	}
	state.UpsertChannel(&st, st.LastInputURL, channel)
	if err := state.Save(st); err != nil {
		return err
	}

	return writeJSON(cmd, map[string]interface{}{
		"ok":         true,
		"flag":       channel.Flag,
		"project_id": channel.ProjectID,
		"channel":    channel,
	})
}

func runFavoriteFill(cmd *cobra.Command, deps Dependencies, project string, limit int) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = 20
	}

	processed := 0
	favorited := 0
	for _, channel := range st.Channels {
		if processed >= limit {
			break
		}
		if !hasUsableContact(channel.Email) {
			continue
		}
		if channel.Flag == "fav" && channel.ProjectID == project {
			continue
		}
		processed++
		if ok, err := applyFlag(cmd.Context(), deps, &st, "fav", channel, project); err != nil {
			if err == config.ErrTokenNotConfigured {
				return fmt.Errorf("%w: run `nanoinf auth token set <token>` first", err)
			}
			return err
		} else if ok {
			favorited++
		}
	}

	if err := state.Save(st); err != nil {
		return err
	}

	return writeJSON(cmd, map[string]interface{}{
		"ok":         true,
		"processed":  processed,
		"favorited":  favorited,
		"project_id": project,
		"limit":      limit,
	})
}

func applyFlag(ctx context.Context, deps Dependencies, st *state.State, flagType string, channel state.Channel, project string) (bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return false, err
	}

	client := api.NewClient(cfg, deps.HTTPClient)
	flagged := map[string]interface{}{
		"id":       channel.ID,
		"platform": channel.Platform,
		"name":     channel.Name,
		"username": channel.Username,
		"icon":     channel.Icon,
		"url":      channel.URL,
		"flag":     flagType,
		"email":    channel.Email,
	}

	groupID := project
	if groupID == "" {
		groupID = channel.ProjectID
	}
	if groupID != "" {
		flagged["group_id"] = groupID
	}

	payload := map[string]interface{}{
		"channels": []map[string]interface{}{flagged},
	}
	if groupID != "" {
		payload["group_id"] = groupID
	}

	resp, err := client.SaveFlag(ctx, payload)
	if err != nil {
		return false, err
	}

	channel.Flag = flagType
	if groupID != "" {
		channel.ProjectID = groupID
	}
	if len(resp.Data.Channels) > 0 {
		if updatedFlag := stringValue(resp.Data.Channels[0]["flag"], ""); updatedFlag != "" {
			channel.Flag = updatedFlag
		}
	}
	state.UpsertChannel(st, st.LastInputURL, channel)
	return true, nil
}
