package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nanoinfluencer/nano-cli/internal/config"
	"github.com/nanoinfluencer/nano-cli/internal/state"
)

func TestRunPrintsReturnedErrors(t *testing.T) {
	t.Setenv(config.ConfigDirEnv, t.TempDir())
	t.Setenv(state.StateDirEnv, t.TempDir())

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := run([]string{"whoami"}, stdout, stderr)

	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "access token not configured") {
		t.Fatalf("expected actionable error on stderr, got %s", stderr.String())
	}
}
