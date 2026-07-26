package runtimeb

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Nene7ko/NeKiro/contracts"
	agentsdk "github.com/Nene7ko/NeKiro/sdks/agent-sdk"
	"github.com/a2aproject/a2a-go/a2a"
)

type runtimeBNestedRecordingInvoker struct {
	context agentsdk.PlatformContext
	request agentsdk.NestedRequest
	result  *agentsdk.NestedResult
}

func (invoker *runtimeBNestedRecordingInvoker) Invoke(_ context.Context, platformContext agentsdk.PlatformContext, request agentsdk.NestedRequest) (*agentsdk.NestedResult, error) {
	invoker.context = platformContext
	invoker.request = request
	return invoker.result, nil
}

func TestNestedServiceBuildsOneRouterOnlyRequestAndPreservesLineage(t *testing.T) {
	config, err := LoadConfig(runtimeBLookup(validRuntimeBEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	invoker := &runtimeBNestedRecordingInvoker{result: &agentsdk.NestedResult{
		InvocationID: "child-1", RootTaskID: "task-1", TraceID: "trace-1", Status: "succeeded", Result: json.RawMessage(`{"agent":"runtime-a"}`),
	}}
	service, err := newNestedService(config, invoker)
	if err != nil {
		t.Fatal(err)
	}
	platformContext := agentsdk.PlatformContext{InvocationID: "root-1", RootTaskID: "task-1", TraceID: "trace-1", WorkspaceID: "workspace-1", AgentID: config.AgentID}
	result, err := service.invoke(t.Context(), platformContext, "reverse-value")
	if err != nil {
		t.Fatal(err)
	}
	if result.InvocationID != "child-1" || invoker.context != platformContext || invoker.request.TargetAgentID != config.TargetAgentID || invoker.request.Capability != config.Capability || invoker.request.Stream {
		t.Fatalf("nested call context=%#v request=%#v result=%#v", invoker.context, invoker.request, result)
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(invoker.request.Input, &input); err != nil || string(input["fixture"]) != `"success"` || string(input["value"]) != `"reverse-value"` {
		t.Fatalf("nested input=%s err=%v", invoker.request.Input, err)
	}
}

func TestNestedServiceRejectsInvalidLineageAndDoesNotRetry(t *testing.T) {
	config, err := LoadConfig(runtimeBLookup(validRuntimeBEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	invoker := &runtimeBNestedRecordingInvoker{result: &agentsdk.NestedResult{
		InvocationID: "child-1", RootTaskID: "wrong-task", TraceID: "trace-1", Status: "succeeded", Result: json.RawMessage(`{"agent":"runtime-a"}`),
	}}
	service, err := newNestedService(config, invoker)
	if err != nil {
		t.Fatal(err)
	}
	platformContext := agentsdk.PlatformContext{InvocationID: "root-1", RootTaskID: "task-1", TraceID: "trace-1", WorkspaceID: "workspace-1", AgentID: config.AgentID}
	if _, err := service.invoke(t.Context(), platformContext, map[string]any{"secret": "value"}); err == nil {
		t.Fatal("invalid nested lineage was accepted")
	}
	if invoker.context != platformContext {
		t.Fatal("nested invoker was not called exactly once before rejecting the returned lineage")
	}
	if safeNestedFailure(errors.New("raw token secret")) == nil {
		t.Fatal("safeNestedFailure returned nil")
	}
}

func TestPlatformContextFromClaimsRequiresExactAgentIdentity(t *testing.T) {
	claims := contracts.RouterInvocationCredentialClaimsV1{AgentID: "runtime-b", InvocationID: "inv-1", RootTaskID: "task-1", TraceID: "trace-1", WorkspaceID: "workspace-1"}
	if _, err := platformContextFromClaims(claims, "runtime-a"); err == nil {
		t.Fatal("wrong Agent identity was accepted")
	}
}

func TestNestedMessageDerivesMissingContextID(t *testing.T) {
	input := &a2a.Message{ID: "root-message"}
	result := &agentsdk.NestedResult{InvocationID: "child-1", Result: json.RawMessage(`{"ok":true}`)}

	message := nestedMessage(input, result)
	if message.ContextID != derivedID("context", input.ID) {
		t.Fatalf("nested message context ID = %q", message.ContextID)
	}
}
