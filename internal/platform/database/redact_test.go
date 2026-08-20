package database

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactRemovesCredentials(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err       error
		forbidden string
		want      string
	}{
		"nil error": {
			err: nil,
		},
		"plain message": {
			err:  errors.New("connection refused"),
			want: "connection refused",
		},
		"connection uri": {
			err:       errors.New("failed to connect to postgres://app:hunter2@db:5433/app"),
			forbidden: "hunter2",
			want:      "redacted",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := redact(tc.err)

			if tc.err == nil {
				if got != nil {
					t.Fatalf("redact(nil) = %v, want nil", got)
				}
				return
			}
			if tc.forbidden != "" && strings.Contains(got.Error(), tc.forbidden) {
				t.Errorf("redact() = %q, want the credential removed", got)
			}
			if !strings.Contains(got.Error(), tc.want) {
				t.Errorf("redact() = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}
