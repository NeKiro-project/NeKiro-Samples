package main

import (
	"testing"

	agenthost "github.com/NeKiro-project/nekiro-sdk-go/agent/host"
)

func TestRunStagesMissingRuntimeBConfiguration(t *testing.T) {
	err := runWithLookup(func(string) (string, bool) { return "", false })
	stage, ok := agenthost.StageOf(err)
	if !ok || stage != agenthost.StageConfig {
		t.Fatalf("StageOf(run error) = %q, %v; error=%v", stage, ok, err)
	}
}
