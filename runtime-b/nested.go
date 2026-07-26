package runtimeb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Nene7ko/NeKiro/contracts"
	agentsdk "github.com/Nene7ko/NeKiro/sdks/agent-sdk"
	"github.com/Nene7ko/NeKiro/sdks/agent-sdk/routerauth"
	"github.com/a2aproject/a2a-go/a2a"
)

type nestedInvoker interface {
	Invoke(context.Context, agentsdk.PlatformContext, agentsdk.NestedRequest) (*agentsdk.NestedResult, error)
}

type nestedService struct {
	config  Config
	invoker nestedInvoker
}

func newNestedService(config Config, invoker nestedInvoker) (*nestedService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if invoker == nil {
		return nil, errors.New("runtime-b nested invoker is required")
	}
	return &nestedService{config: config, invoker: invoker}, nil
}

func (service *nestedService) platformContext(ctx context.Context) (agentsdk.PlatformContext, error) {
	claims, ok := routerauth.ClaimsFromContext(ctx)
	if !ok {
		return agentsdk.PlatformContext{}, invalidParams("managed Router credential context is required")
	}
	platformContext, err := platformContextFromClaims(claims, service.config.AgentID)
	if err != nil {
		return agentsdk.PlatformContext{}, invalidParams(err.Error())
	}
	return platformContext, nil
}

func (service *nestedService) invoke(ctx context.Context, platformContext agentsdk.PlatformContext, value any) (*agentsdk.NestedResult, error) {
	encodedValue, err := json.Marshal(value)
	if err != nil {
		return nil, invalidParams("value must be JSON-compatible")
	}
	input, err := json.Marshal(map[string]json.RawMessage{
		"fixture": json.RawMessage(`"success"`),
		"value":   encodedValue,
	})
	if err != nil {
		return nil, errors.New("runtime-b encode nested input failed")
	}
	result, err := service.invoker.Invoke(ctx, platformContext, agentsdk.NestedRequest{
		TargetAgentID: service.config.TargetAgentID,
		Capability:    service.config.Capability,
		Input:         input,
		Stream:        false,
	})
	if err != nil {
		return nil, err
	}
	if result == nil || result.InvocationID == "" || result.RootTaskID == "" || result.TraceID == "" || result.Status != "succeeded" || result.Result == nil {
		return nil, errors.New("runtime-b nested result is incomplete")
	}
	if result.InvocationID == platformContext.InvocationID || result.RootTaskID != platformContext.RootTaskID || result.TraceID != platformContext.TraceID {
		return nil, errors.New("runtime-b nested result lineage is invalid")
	}
	return result, nil
}

func nestedMessage(input *a2a.Message, result *agentsdk.NestedResult) *a2a.Message {
	contextID := input.ContextID
	if contextID == "" {
		contextID = derivedID("context", input.ID)
	}
	return &a2a.Message{
		ID:        "runtime-b-nested-result-" + input.ID,
		ContextID: contextID,
		Role:      a2a.MessageRoleAgent,
		Parts: []a2a.Part{a2a.DataPart{Data: map[string]any{
			"agent":             "runtime-b",
			"fixture":           string(fixtureNested),
			"childInvocationId": result.InvocationID,
			"childResult":       result.Result,
		}}},
	}
}

func safeNestedFailure(err error) error {
	var routerError *agentsdk.RouterError
	if errors.As(err, &routerError) {
		return fmt.Errorf("runtime-b nested Router failure: %s", routerError.Code)
	}
	return errors.New("runtime-b nested invocation failure (unknown category)")
}

func platformContextFromClaims(claims contracts.RouterInvocationCredentialClaimsV1, agentID string) (agentsdk.PlatformContext, error) {
	if claims.AgentID != agentID {
		return agentsdk.PlatformContext{}, errors.New("managed Router credential Agent identity is invalid")
	}
	platformContext := agentsdk.PlatformContext{
		InvocationID: claims.InvocationID,
		RootTaskID:   claims.RootTaskID,
		TraceID:      string(claims.TraceID),
		WorkspaceID:  claims.WorkspaceID,
		AgentID:      claims.AgentID,
	}
	if err := platformContext.Validate(); err != nil {
		return agentsdk.PlatformContext{}, err
	}
	return platformContext, nil
}
