// Package git provides a restricted native-Git adapter for public repository topology.
package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var ErrUnsafeRepository = errors.New("unsafe Git repository")

type Runner struct {
	Executable string
	Root       string
	Timeout    time.Duration
	MaxOutput  int64
}

func (runner Runner) Fetch(ctx context.Context, canonicalURL string, projectID int64) error {
	if projectID <= 0 || !strings.HasPrefix(canonicalURL, "https://") ||
		strings.ContainsAny(canonicalURL, "\r\n") || strings.Contains(canonicalURL, "@") {
		return ErrUnsafeRepository
	}
	root, err := filepath.Abs(runner.Root)
	if err != nil || root == "/" || root == "." || root == "" {
		return ErrUnsafeRepository
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("prepare Git mirror root: %w", err)
	}
	destination := filepath.Join(root, fmt.Sprintf("project-%d.git", projectID))
	if relative, relativeErr := filepath.Rel(root, destination); relativeErr != nil ||
		relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ErrUnsafeRepository
	}
	timeout := runner.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	executable := runner.Executable
	if executable == "" {
		executable = "git"
	}
	arguments := []string{
		"-c", "credential.helper=", "-c", "core.hooksPath=/dev/null",
		"-c", "protocol.file.allow=never", "-c", "protocol.ext.allow=never",
		"-c", "submodule.recurse=false", "clone", "--bare", "--no-tags", "--filter=blob:none",
		"--", canonicalURL, destination,
	}
	command := exec.CommandContext(commandCtx, executable, arguments...)
	command.Dir = root
	command.Env = []string{
		"HOME=" + root,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GIT_PROTOCOL_FROM_USER=0",
		"PATH=/usr/bin:/bin",
	}
	limit := runner.MaxOutput
	if limit <= 0 {
		limit = 64 << 10
	}
	output := &limitedWriter{remaining: limit}
	command.Stdout = output
	command.Stderr = output
	if err := command.Run(); err != nil {
		return fmt.Errorf("restricted Git fetch failed: %w", err)
	}
	return nil
}

type limitedWriter struct {
	remaining int64
}

func (writer *limitedWriter) Write(body []byte) (int, error) {
	if writer.remaining <= 0 {
		return len(body), nil
	}
	written, err := io.Discard.Write(body[:min(int64(len(body)), writer.remaining)])
	writer.remaining -= int64(written)
	return len(body), err
}
