package runtimeb

import "testing"

func validRuntimeBEnvironment() map[string]string {
	return map[string]string{
		AgentIDEnvironment:                         "agent-runtime-b",
		RouterEnvironment:                          "http://127.0.0.1:4101",
		RouterTokenEnvironment:                     "opaque-token",
		TargetAgentEnvironment:                     "agent-runtime-a",
		CapabilityEnvironment:                      "runtime.echo",
		ResponseLimitEnvironment:                   "1048576",
		EventLimitEnvironment:                      "1048576",
		"NEKIRO_AGENT_ROUTER_ISSUER":               "https://a2a-router.nekiro.test",
		"NEKIRO_AGENT_ROUTER_AUDIENCE":             "http://runtime-b:8092",
		"NEKIRO_AGENT_ROUTER_KEY_ID":               "router-key-1",
		"NEKIRO_AGENT_ROUTER_PUBLIC_KEY_BASE64URL": "A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg",
	}
}

func runtimeBLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, exists := values[name]
		return value, exists
	}
}

func TestLoadConfigRequiresAndValidatesAllSettings(t *testing.T) {
	config, err := LoadConfig(runtimeBLookup(validRuntimeBEnvironment()))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.AgentID != "agent-runtime-b" || config.TargetAgentID != "agent-runtime-a" || config.ResponseLimit != 1048576 || config.EventLimit != 1048576 {
		t.Fatalf("LoadConfig() = %+v", config)
	}

	for name := range validRuntimeBEnvironment() {
		environment := validRuntimeBEnvironment()
		delete(environment, name)
		if _, err := LoadConfig(runtimeBLookup(environment)); err == nil {
			t.Errorf("missing %s was accepted", name)
		}
	}
}

func TestLoadConfigRejectsInvalidValuesWithoutDefaults(t *testing.T) {
	tests := map[string]string{
		AgentIDEnvironment:       " agent-runtime-b",
		RouterEnvironment:        "localhost:4101",
		RouterTokenEnvironment:   " opaque-token",
		TargetAgentEnvironment:   "agent runtime-a",
		CapabilityEnvironment:    "runtime/echo",
		ResponseLimitEnvironment: "+1",
		EventLimitEnvironment:    "2147483648",
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			environment := validRuntimeBEnvironment()
			environment[name] = value
			if _, err := LoadConfig(runtimeBLookup(environment)); err == nil {
				t.Fatalf("invalid %s=%q was accepted", name, value)
			}
		})
	}
	for _, value := range []string{"http://127.0.0.1:4101/", "http://127.0.0.1:4101?", "http://127.0.0.1:65536", "http://127.0.0.1:4101#", "http://a2a-router:"} {
		environment := validRuntimeBEnvironment()
		environment[RouterEnvironment] = value
		if _, err := LoadConfig(runtimeBLookup(environment)); err == nil {
			t.Fatalf("Router URL %q was accepted", value)
		}
	}
}

func TestConfigValidateRejectsValuesBypassedAroundEnvironmentLoader(t *testing.T) {
	config, err := LoadConfig(runtimeBLookup(validRuntimeBEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	config.TargetAgentID = "agent runtime-a"
	if err := config.Validate(); err == nil {
		t.Fatal("Config.Validate accepted an invalid target Agent ID")
	}
}
