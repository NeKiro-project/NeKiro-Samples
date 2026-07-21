package runtimea

import (
	"testing"
)

func validEnvironment() map[string]string {
	return map[string]string{
		ListenAddressEnvironment: "127.0.0.1:4103",
		AgentIDEnvironment:       "agent-runtime-a",
		RouterEnvironment:        "http://127.0.0.1:4101",
		RouterTokenEnvironment:   "opaque-token",
		TargetAgentEnvironment:   "agent-runtime-b",
		CapabilityEnvironment:    "fixture",
		ResponseLimitEnvironment: "1048576",
		EventLimitEnvironment:    "1048576",
	}
}

func lookupEnvironment(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, exists := values[name]
		return value, exists
	}
}

func TestLoadConfigRequiresAndValidatesAllSettings(t *testing.T) {
	config, err := LoadConfig(lookupEnvironment(validEnvironment()))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.AgentID != "agent-runtime-a" || config.TargetAgentID != "agent-runtime-b" || config.ResponseLimit != 1048576 || config.EventLimit != 1048576 {
		t.Fatalf("LoadConfig() = %+v", config)
	}

	for name := range validEnvironment() {
		environment := validEnvironment()
		delete(environment, name)
		if _, err := LoadConfig(lookupEnvironment(environment)); err == nil {
			t.Errorf("missing %s was accepted", name)
		}
	}
}

func TestLoadConfigRejectsInvalidValuesWithoutDefaults(t *testing.T) {
	tests := map[string]string{
		ListenAddressEnvironment: "127.0.0.1:0",
		AgentIDEnvironment:       " agent-runtime-a",
		RouterEnvironment:        "localhost:4101",
		RouterTokenEnvironment:   " opaque-token",
		TargetAgentEnvironment:   "agent runtime-b",
		CapabilityEnvironment:    "fixture/extra",
		ResponseLimitEnvironment: "+1",
		EventLimitEnvironment:    "2147483648",
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			environment := validEnvironment()
			environment[name] = value
			if _, err := LoadConfig(lookupEnvironment(environment)); err == nil {
				t.Fatalf("invalid %s=%q was accepted", name, value)
			}
		})
	}
}
