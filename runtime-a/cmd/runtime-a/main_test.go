package main

import (
	"testing"

	agenthost "github.com/NeKiro-project/nekiro-sdk-go/agent/host"
	registrationnacos "github.com/NeKiro-project/nekiro-sdk-go/agent/registration/nacos"
)

func TestRunStagesMissingRuntimeAConfiguration(t *testing.T) {
	err := runWithLookup(func(string) (string, bool) { return "", false })
	stage, ok := agenthost.StageOf(err)
	if !ok || stage != agenthost.StageConfig {
		t.Fatalf("StageOf(run error) = %q, %v; error=%v", stage, ok, err)
	}
}

func TestNewRuntimeRegistrationUsesPublicSDKModes(t *testing.T) {
	registration, readiness, err := newRuntimeRegistration(registrationnacos.Config{
		Mode: registrationnacos.ModeDisabled, AgentID: "runtime-a", InstanceID: "runtime-a-primary",
	})
	if err != nil || registration != nil || !readiness.Ready() {
		t.Fatalf("disabled registration=%v readiness=%v error=%v", registration, readiness, err)
	}
	registration, readiness, err = newRuntimeRegistration(registrationnacos.Config{Mode: registrationnacos.ModeNacos})
	if err == nil || registration != nil || readiness != nil {
		t.Fatalf("invalid Nacos registration=%v readiness=%v error=%v", registration, readiness, err)
	}
}
