package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

func TestStructuredPlannerRejectsUnknownAndMultipleDocuments(t *testing.T) {
	planner := StructuredPlanner{}
	valid := `{"action":"repository.add","repository":{"project_id":"1","project_version":1,"url":"https://github.com/acme/sdk","role":"sdk"},"effect":"Attach one SDK repository","quota_name":"project_repositories","quota_cost":1,"quota_limit":20,"quota_used":1,"action_count":1}`
	if _, err := planner.Plan(context.Background(), access.Principal{}, valid); err != nil {
		t.Fatalf("valid typed draft: %v", err)
	}
	for _, value := range []string{valid[:len(valid)-1] + `,"sql":"SELECT 1"}`, valid + valid} {
		if _, err := planner.Plan(context.Background(), access.Principal{}, value); !errors.Is(err, ErrInvalid) {
			t.Fatalf("unsafe structured draft error = %v", err)
		}
	}
}

func TestBoundedPlannerRejectsUnboundedConfigurationAndOutput(t *testing.T) {
	if _, err := NewBoundedPlanner(StructuredPlanner{}, Limits{}); !errors.Is(err, ErrRunLimit) {
		t.Fatalf("unbounded limits error = %v", err)
	}
	planner, err := NewBoundedPlanner(StructuredPlanner{}, Limits{
		MaxSteps: 1, Timeout: time.Second, MaxOutputBytes: 8, MaxCostMicros: 1, Concurrency: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	valid := `{"action":"repository.add","repository":{"project_id":"1","project_version":1,"url":"https://github.com/acme/sdk","role":"sdk"},"effect":"Attach one SDK repository","quota_name":"project_repositories","quota_cost":1,"quota_limit":20,"quota_used":1,"action_count":1}`
	if _, err := planner.Plan(context.Background(), access.Principal{}, valid); !errors.Is(err, ErrRunLimit) {
		t.Fatalf("oversized output error = %v", err)
	}
}
