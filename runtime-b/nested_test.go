package runtimeb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/NeKiro-project/NeKiro/contracts"
	agentsdk "github.com/NeKiro-project/NeKiro/sdks/agent-sdk"
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

func TestNestedFixtureUsesManagedRouterContext(t *testing.T) {
	if result, err := NewHandler().OnSendMessage(t.Context(), fixtureParams("unconfigured-nested", fixtureNested, "value")); result != nil || !errors.Is(err, a2a.ErrInvalidParams) {
		t.Fatalf("unconfigured nested fixture = (%#v, %v)", result, err)
	}

	environment := validRuntimeBEnvironment()
	environment[AgentIDEnvironment] = "runtime-b"
	config, err := LoadConfig(runtimeBLookup(environment))
	if err != nil {
		t.Fatal(err)
	}
	invoker := &runtimeBNestedRecordingInvoker{result: &agentsdk.NestedResult{
		InvocationID: "child-1", RootTaskID: "task-runtime-b", TraceID: "trace-runtime-b", Status: "succeeded", Result: json.RawMessage(`{"agent":"runtime-a","value":"ok"}`),
	}}
	service, err := newNestedService(config, invoker)
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{tasks: make(map[a2a.TaskID]*runtimeTask), agentID: config.AgentID, nested: service}
	server := httptest.NewServer(httpHandler(t, handler))
	t.Cleanup(server.Close)

	client := newA2AClient(t, server, nil)
	result, err := client.SendMessage(t.Context(), fixtureParams("nested-message", fixtureNested, "reverse-value"))
	if err != nil {
		t.Fatalf("nested send: %v", err)
	}
	message := requireMessage(t, result)
	data := requireDataPart(t, message.Parts[0])
	if message.ContextID != derivedID("context", "nested-message") || data.Data["agent"] != "runtime-b" || data.Data["childInvocationId"] != "child-1" {
		t.Fatalf("nested response = %#v context=%q", data.Data, message.ContextID)
	}
	if invoker.context.InvocationID != "inv_runtime_b" || invoker.context.RootTaskID != "task-runtime-b" || invoker.context.TraceID != "trace-runtime-b" || invoker.context.WorkspaceID != "workspace-a" || invoker.context.AgentID != config.AgentID || invoker.request.TargetAgentID != config.TargetAgentID || invoker.request.Capability != config.Capability || invoker.request.Stream {
		t.Fatalf("nested managed call context=%#v request=%#v", invoker.context, invoker.request)
	}
}
