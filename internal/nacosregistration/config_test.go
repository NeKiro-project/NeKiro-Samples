package nacosregistration

import (
	"path/filepath"
	"testing"
)

func TestLoadRequiresExactReleaseAndExplicitFreshness(t *testing.T) {
	values := validEnvironment()
	config, err := Load(mapLookup(values), "RUNTIME_B", "runtime-b", "runtime-b-primary")
	if err != nil {
		t.Fatal(err)
	}
	if config.ReleaseID != "rel_runtime_b_1" || config.PortName != "a2a" || config.HeartbeatTimeout.Milliseconds() != 5000 || config.IPDeleteTimeout.Milliseconds() != 10000 {
		t.Fatalf("config=%#v", config)
	}
	for _, name := range []string{
		"RUNTIME_B_RELEASE_ID", "RUNTIME_B_CARD_DIGEST", "RUNTIME_B_CANONICAL_ENDPOINT", "RUNTIME_B_AUDIENCE",
		"RUNTIME_B_NACOS_PORT_NAME", "RUNTIME_B_NACOS_WEIGHT", "RUNTIME_B_NACOS_HEARTBEAT_TIMEOUT_MS", "RUNTIME_B_NACOS_IP_DELETE_TIMEOUT_MS",
	} {
		invalid := validEnvironment()
		delete(invalid, name)
		if _, err := Load(mapLookup(invalid), "RUNTIME_B", "runtime-b", "runtime-b-primary"); err == nil {
			t.Errorf("missing %s was accepted", name)
		}
	}
}

func TestLoadRequiresExplicitHTTPSRegistrationTrust(t *testing.T) {
	values := validEnvironment()
	values["RUNTIME_B_NACOS_API_ORIGIN"] = "https://nacos.internal:8848/nacos"
	values["RUNTIME_B_NACOS_TLS_CA_FILE"] = filepath.Join(t.TempDir(), "ca.pem")
	values["RUNTIME_B_NACOS_TLS_SERVER_NAME"] = "nacos.internal"
	config, err := Load(mapLookup(values), "RUNTIME_B", "runtime-b", "runtime-b-primary")
	if err != nil || config.TLSCAFile == "" || config.TLSServerName != "nacos.internal" {
		t.Fatalf("HTTPS config=%#v error=%v", config, err)
	}

	for name, mutate := range map[string]func(map[string]string){
		"missing CA":          func(values map[string]string) { delete(values, "RUNTIME_B_NACOS_TLS_CA_FILE") },
		"missing server name": func(values map[string]string) { delete(values, "RUNTIME_B_NACOS_TLS_SERVER_NAME") },
		"relative CA":         func(values map[string]string) { values["RUNTIME_B_NACOS_TLS_CA_FILE"] = "ca.pem" },
		"invalid server name": func(values map[string]string) { values["RUNTIME_B_NACOS_TLS_SERVER_NAME"] = "nacos_internal" },
		"client cert only": func(values map[string]string) {
			values["RUNTIME_B_NACOS_TLS_CLIENT_CERT_FILE"] = filepath.Join(t.TempDir(), "client.pem")
		},
		"client key only": func(values map[string]string) {
			values["RUNTIME_B_NACOS_TLS_CLIENT_KEY_FILE"] = filepath.Join(t.TempDir(), "client-key.pem")
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := make(map[string]string, len(values))
			for key, value := range values {
				invalid[key] = value
			}
			mutate(invalid)
			if _, err := Load(mapLookup(invalid), "RUNTIME_B", "runtime-b", "runtime-b-primary"); err == nil {
				t.Fatal("invalid HTTPS registration trust was accepted")
			}
		})
	}
}

func TestLoadRejectsTLSFieldsForHTTPAndDisabledRegistration(t *testing.T) {
	for _, mode := range []string{"http", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			values := validEnvironment()
			if mode == "disabled" {
				values = map[string]string{"RUNTIME_B_REGISTRATION_MODE": ModeDisabled}
			}
			values["RUNTIME_B_NACOS_TLS_CA_FILE"] = filepath.Join(t.TempDir(), "ca.pem")
			if _, err := Load(mapLookup(values), "RUNTIME_B", "runtime-b", "runtime-b-primary"); err == nil {
				t.Fatal("non-HTTPS registration accepted TLS fields")
			}
		})
	}
}

func TestLoadRejectsMismatchedTargetAndFreshnessOrder(t *testing.T) {
	for name, mutate := range map[string]func(map[string]string){
		"audience":          func(values map[string]string) { values["RUNTIME_B_AUDIENCE"] = "http://runtime-a:8091" },
		"heartbeat timeout": func(values map[string]string) { values["RUNTIME_B_NACOS_HEARTBEAT_TIMEOUT_MS"] = "1000" },
		"delete timeout":    func(values map[string]string) { values["RUNTIME_B_NACOS_IP_DELETE_TIMEOUT_MS"] = "5000" },
	} {
		t.Run(name, func(t *testing.T) {
			values := validEnvironment()
			mutate(values)
			if _, err := Load(mapLookup(values), "RUNTIME_B", "runtime-b", "runtime-b-primary"); err == nil {
				t.Fatal("invalid registration config was accepted")
			}
		})
	}
}

func validEnvironment() map[string]string {
	return map[string]string{
		"RUNTIME_B_REGISTRATION_MODE": "nacos", "RUNTIME_B_AGENT_CARD_VERSION": "1.0.0", "RUNTIME_B_RELEASE_ID": "rel_runtime_b_1",
		"RUNTIME_B_CARD_DIGEST":        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"RUNTIME_B_CANONICAL_ENDPOINT": "http://runtime-b:8092/", "RUNTIME_B_AUDIENCE": "http://runtime-b:8092",
		"RUNTIME_B_NACOS_API_ORIGIN": "http://nacos:8848/nacos", "RUNTIME_B_NACOS_NAMESPACE_ID": "public",
		"RUNTIME_B_NACOS_GROUP_NAME": "NEKIRO", "RUNTIME_B_NACOS_SERVICE_NAME": "runtime-b", "RUNTIME_B_NACOS_CLUSTER_NAME": "DEFAULT",
		"RUNTIME_B_NACOS_PORT_NAME": "a2a", "RUNTIME_B_NACOS_ADVERTISED_IP": "127.0.0.1", "RUNTIME_B_NACOS_ADVERTISED_PORT": "8092",
		"RUNTIME_B_NACOS_WEIGHT": "1", "RUNTIME_B_NACOS_HEARTBEAT_INTERVAL_MS": "1000", "RUNTIME_B_NACOS_HEARTBEAT_TIMEOUT_MS": "5000",
		"RUNTIME_B_NACOS_IP_DELETE_TIMEOUT_MS": "10000", "RUNTIME_B_NACOS_REQUEST_TIMEOUT_MS": "1000", "RUNTIME_B_NACOS_AUTH_MODE": "none",
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
