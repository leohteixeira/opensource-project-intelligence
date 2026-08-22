package project_test

import (
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

func TestRepositoryAndLifecycleRules(t *testing.T) {
	t.Parallel()
	base := mustProject(t)

	tests := []struct {
		id   string
		run  func(project.Project) error
		want error
	}{
		{"UT-043 unsupported role", func(value project.Project) error {
			_, err := value.AddRepository(project.Repository{ID: 3, ProjectID: 1, CanonicalURL: "https://github.com/acme/sdk", Role: "invalid"}, 10)
			return err
		}, project.ErrInvalid},
		{"UT-044 cannot lose primary", func(value project.Project) error {
			_, err := value.ChangeRepositoryRole(2, project.RoleCore)
			return err
		}, project.ErrConflict},
		{"UT-045 repository limit", func(value project.Project) error {
			_, err := value.AddRepository(project.Repository{ID: 3, ProjectID: 1, CanonicalURL: "https://github.com/acme/sdk", Role: project.RoleSDK}, 1)
			return err
		}, project.ErrConflict},
		{"UT-047 repeated attachment", func(value project.Project) error {
			got, err := value.AddRepository(value.Repositories[0], 10)
			if err == nil && len(got.Repositories) != 1 {
				return errors.New("duplicate repository was attached")
			}
			return err
		}, nil},
		{"UT-048 replacement validates first", func(value project.Project) error {
			_, err := value.ChangeRepositoryRole(999, project.RolePrimary)
			if !errors.Is(err, project.ErrNotFound) || value.Repositories[0].Role != project.RolePrimary {
				return errors.New("old primary changed before replacement validation")
			}
			return nil
		}, nil},
		{"UT-049 archived repository read-only", func(value project.Project) error {
			value.State = project.StateArchived
			_, err := value.AddRepository(project.Repository{ID: 3, ProjectID: 1, CanonicalURL: "https://github.com/acme/sdk", Role: project.RoleSDK}, 10)
			return err
		}, project.ErrConflict},
		{"UT-057 unknown transition", func(value project.Project) error {
			_, err := value.Transition("unknown", value.Version, true)
			return err
		}, project.ErrConflict},
		{"UT-060 lifecycle admin only", func(value project.Project) error {
			_, err := value.Transition(project.StatePaused, value.Version, false)
			return err
		}, project.ErrPermissionDenied},
		{"UT-061 repeated pause idempotent", func(value project.Project) error {
			paused, err := value.Transition(project.StatePaused, value.Version, true)
			if err != nil {
				return err
			}
			repeated, err := paused.Transition(project.StatePaused, paused.Version, true)
			if err == nil && repeated.Version != paused.Version {
				return errors.New("idempotent transition changed version")
			}
			return err
		}, nil},
		{"UT-062 deleted cannot restore", func(value project.Project) error {
			value.State = project.StateDeleted
			_, err := value.Transition(project.StateActive, value.Version, true)
			return err
		}, project.ErrConflict},
		{"UT-063 archived rejects sync", func(value project.Project) error {
			value.State = project.StateArchived
			return value.CanSynchronize()
		}, project.ErrConflict},
		{"UT-070 paused rejects sync", func(value project.Project) error {
			value.State = project.StatePaused
			return value.CanSynchronize()
		}, project.ErrConflict},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			err := test.run(base)
			if !errors.Is(err, test.want) {
				t.Fatalf("got error %v, want %v", err, test.want)
			}
		})
	}
}

func TestHistoryAndSourceRules(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

	t.Run("UT-065 no checkpoint begins 180-day backfill", func(t *testing.T) {
		rangeValue := project.InitialHistoryRange(now, 0)
		if days := int(rangeValue.To.Sub(rangeValue.From).Hours() / 24); days != 180 {
			t.Fatalf("got %d days", days)
		}
	})
	t.Run("UT-071 reversed and future-only ranges rejected", func(t *testing.T) {
		if _, err := project.ValidateHistoryRange(now, now.Add(-24*time.Hour), now, 365); !errors.Is(err, project.ErrInvalid) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("UT-073 provider maximum is explicit", func(t *testing.T) {
		if _, err := project.ValidateHistoryRange(now.AddDate(-2, 0, 0), now, now, 365); !errors.Is(err, project.ErrInvalid) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("UT-084 private transition marks unavailable", func(t *testing.T) {
		source, err := (project.Source{ID: 1, ProjectID: 1, State: project.SourceAvailable, Public: true, Version: 1}).MarkUnavailable("visibility changed")
		if err != nil || source.State != project.SourceUnavailable || source.Public || source.NextRunAt != nil {
			t.Fatalf("got %#v, %v", source, err)
		}
	})
}

func TestAssociationCorrectionIsRetainedAndIdempotent(t *testing.T) {
	t.Parallel()
	association := project.Association{ID: 1, SourceID: 2, ProjectID: 3, Status: project.AssociationLinked, Version: 1}
	correction := project.Correction{Action: "split", Reason: "different product", ActorID: 4, At: time.Now()}

	corrected, enqueue, err := association.Correct(correction)
	if err != nil || !enqueue || corrected.Constraint == nil {
		t.Fatalf("UT-055 correction failed: %#v %v", corrected, err)
	}
	repeated, enqueue, err := corrected.Correct(correction)
	if err != nil || enqueue || repeated.Version != corrected.Version {
		t.Fatalf("UT-054 repeated correction was not idempotent: %#v %v", repeated, err)
	}
}

func mustProject(t *testing.T) project.Project {
	t.Helper()
	value, err := project.New(1, 1, "Acme", "acme", project.Repository{
		ID: 2, ProjectID: 1, Provider: "github", CanonicalURL: "https://github.com/acme/core", Role: project.RolePrimary,
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}
