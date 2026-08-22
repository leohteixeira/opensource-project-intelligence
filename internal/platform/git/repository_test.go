package git_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	gitadapter "github.com/leohteixeira/opensource-project-intelligence/internal/platform/git"
)

func TestRestrictedGitTransport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runner := gitadapter.Runner{Root: root, Timeout: time.Second}
	for _, value := range []string{"file:///etc/passwd", "ssh://git@example.com/project", "https://token@example.com/project"} {
		if err := runner.Fetch(context.Background(), value, 1); !errors.Is(err, gitadapter.ErrUnsafeRepository) {
			t.Fatalf("UT-255 %q got %v", value, err)
		}
	}
}

func TestRestrictedGitUsesArgumentVectorAndSafeEnvironment(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "fake-git")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := gitadapter.Runner{Executable: executable, Root: filepath.Join(root, "mirrors"), Timeout: time.Second}
	if err := runner.Fetch(context.Background(), "https://example.com/acme/project", 9); err != nil {
		t.Fatalf("IT-116 safe controlled fetch failed: %v", err)
	}
}
