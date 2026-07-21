package runtimea

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	agentsdk "github.com/Nene7ko/NeKiro/sdks/agent-sdk"
)

func TestRuntimeAConcurrentCallsRemainIsolated(t *testing.T) {
	config, err := LoadConfig(lookupEnvironment(validEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	invoker := &recordingInvoker{result: func(contextValue agentsdk.PlatformContext) *agentsdk.NestedResult {
		return &agentsdk.NestedResult{
			InvocationID: "child-" + contextValue.InvocationID,
			RootTaskID:   contextValue.RootTaskID,
			TraceID:      contextValue.TraceID,
			Status:       "succeeded",
			Result:       json.RawMessage(`{"agent":"runtime-b"}`),
		}
	}}
	service, err := newNestedService(config, invoker)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := newRuntimeEngine(config, service)
	if err != nil {
		t.Fatal(err)
	}
	const calls = 100
	errors := make(chan error, calls)
	var waitGroup sync.WaitGroup
	for index := 0; index < calls; index++ {
		index := index
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			invocationID := "root-" + string(rune('a'+index%26)) + "-" + string(rune('0'+index/26))
			platformContext := agentsdk.PlatformContext{InvocationID: invocationID, RootTaskID: "task-" + invocationID, TraceID: "trace-" + invocationID, WorkspaceID: "workspace-" + invocationID, AgentID: config.AgentID}
			result, err := engine.run(context.Background(), platformContext, []byte(`{"fixture":"success","value":"isolated"}`))
			if err != nil {
				errors <- err
				return
			}
			var combined map[string]any
			if err := json.Unmarshal(result, &combined); err != nil || combined["childInvocationId"] != "child-"+invocationID {
				errors <- errIsolationMismatch
			}
		}()
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	invoker.mu.Lock()
	defer invoker.mu.Unlock()
	if len(invoker.calls) != calls {
		t.Fatalf("nested call count = %d, want %d", len(invoker.calls), calls)
	}
}

var errIsolationMismatch = isolationError("runtime-a result context crossed calls")

type isolationError string

func (err isolationError) Error() string { return string(err) }

func TestRuntimeABoundaryDoesNotCopyCredentialIntoOutputOrPlatformModule(t *testing.T) {
	config, err := LoadConfig(lookupEnvironment(validEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(config.RouterToken, config.AgentID) {
		t.Fatal("test credential unexpectedly overlaps identity")
	}
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "trpc.group/trpc-go/trpc-agent-go v1.10.0") {
		t.Fatal("Runtime A framework is not pinned in the nested module")
	}
	if strings.Contains(string(data), "apps/") || strings.Contains(string(data), "agents/runtime-b") {
		t.Fatal("nested module declares a platform or Runtime B dependency")
	}
	for _, sourceName := range []string{"config.go", "nested.go", "runtime.go", "handler.go"} {
		source, err := os.ReadFile(sourceName)
		if err != nil {
			t.Fatal(err)
		}
		text := string(source)
		for _, forbidden := range []string{"log.Print", "log.Printf", "log.Fatal", "fmt.Print", "fmt.Println"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s emits unbounded content through %s", sourceName, forbidden)
			}
		}
	}
}
