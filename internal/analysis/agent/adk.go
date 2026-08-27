package agent

import (
	"context"
	"errors"

	adkagent "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	adktool "google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
)

const proposalToolName = "propose_repository_add"

type ProposalTool interface {
	Draft(context.Context, access.Principal, RepositoryAdd) (Draft, error)
}

// NewADK constructs the complete ADK boundary. It exposes one typed proposal
// tool and deliberately provides no SQL, filesystem, object-store, provider
// SDK, credential, broker, or arbitrary HTTP capability.
func NewADK(llm model.LLM, principal access.Principal, proposals ProposalTool) (adkagent.Agent, error) {
	if llm == nil || proposals == nil {
		return nil, errors.New("ADK model and proposal port are required")
	}
	tool, err := functiontool.New(functiontool.Config{
		Name:                proposalToolName,
		Description:         "Prepare one non-destructive repository.add proposal for later application confirmation.",
		RequireConfirmation: true,
	}, func(ctx adkagent.Context, input RepositoryAdd) (Draft, error) {
		return proposals.Draft(ctx, principal, input)
	})
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "bounded_project_assistant",
		Description: "Answers project questions and may prepare one repository.add proposal.",
		Model:       llm,
		Instruction: "Use only the supplied typed tool. Never request or infer credentials, membership, policy, lifecycle, archive, deletion, or multiple actions.",
		Tools:       []adktool.Tool{tool},
	})
}
