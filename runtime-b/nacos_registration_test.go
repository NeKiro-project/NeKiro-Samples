package runtimeb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNacosRegistrationOwnsRegisterHeartbeatAndDeregister(t *testing.T) {
	var mu sync.Mutex
	methods := make([]string, 0, 3)
	heartbeat := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		methods = append(methods, request.Method+" "+request.URL.Path)
		mu.Unlock()
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		query := request.Form
		if query.Get("serviceName") != "NEKIRO@@runtime-b" || query.Get("groupName") != "NEKIRO" || query.Get("clusterName") != "DEFAULT" || query.Get("namespaceId") != "public" || query.Get("ip") != "127.0.0.1" || query.Get("port") != "8092" {
			t.Errorf("request query=%v", query)
		}
		if request.Method == http.MethodPost {
			var metadata map[string]string
			if json.Unmarshal([]byte(query.Get("metadata")), &metadata) != nil || metadata["nekiro.instanceId"] != "runtime-b-directory" || metadata["preserved.heart.beat.interval"] != "1000" || metadata["preserved.heart.beat.timeout"] != "5000" || metadata["preserved.ip.delete.timeout"] != "10000" || query.Get("ephemeral") != "true" || query.Get("weight") != "1" {
				t.Errorf("registration metadata=%v query=%v", metadata, query)
			}
		}
		if request.Method == http.MethodPut {
			select {
			case heartbeat <- struct{}{}:
			default:
			}
			var beat struct {
				Service  string            `json:"serviceName"`
				Metadata map[string]string `json:"metadata"`
				Weight   float64           `json:"weight"`
			}
			if json.Unmarshal([]byte(query.Get("beat")), &beat) != nil || beat.Service != "NEKIRO@@runtime-b" || beat.Metadata["nekiro.instanceId"] != "runtime-b-directory" || beat.Metadata["preserved.heart.beat.timeout"] != "5000" || beat.Weight != 1.0 || query.Get("ephemeral") != "true" {
				t.Errorf("heartbeat=%v query=%v", beat, query)
			}
		}
		if request.Method == http.MethodPut {
			_, _ = writer.Write([]byte(`{"clientBeatInterval":1000,"code":10200,"lightBeatEnabled":true}`))
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registration := testRegistration(t, server, time.Second)
	if err := registration.Register(t.Context()); err != nil || !registration.Ready() {
		t.Fatalf("Register ready=%v error=%v", registration.Ready(), err)
	}
	runContext, cancelRun := context.WithCancel(t.Context())
	runErrors := make(chan error, 1)
	go func() { runErrors <- registration.Run(runContext) }()
	select {
	case <-heartbeat:
	case <-time.After(3 * time.Second):
		t.Fatal("heartbeat was not sent")
	}
	cancelRun()
	if err := <-runErrors; err != nil {
		t.Fatal(err)
	}
	if err := registration.Deregister(t.Context()); err != nil || registration.Ready() {
		t.Fatalf("Deregister ready=%v error=%v", registration.Ready(), err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{"POST /nacos/v1/ns/instance", "PUT /nacos/v1/ns/instance/beat", "DELETE /nacos/v1/ns/instance"}
	if len(methods) != len(want) {
		t.Fatalf("methods=%v", methods)
	}
	for index := range want {
		if methods[index] != want[index] {
			t.Fatalf("methods=%v want=%v", methods, want)
		}
	}
}

func TestNacosHeartbeatFailureMakesRuntimeNotReadyWithoutRetry(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method == http.MethodPut {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registration := testRegistration(t, server, time.Second)
	if err := registration.Register(t.Context()); err != nil {
		t.Fatal(err)
	}
	err := registration.Run(t.Context())
	if err == nil || registration.Ready() || requests != 2 {
		t.Fatalf("Run error=%v ready=%v requests=%d", err, registration.Ready(), requests)
	}
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	handler := NewHandler()
	application := httpHandlerWithReadiness(t, handler, registration)
	application.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status=%d", response.Code)
	}
}

func TestNacosRegistrationClassifiesCanceledAndUnavailableRequests(t *testing.T) {
	registration, err := NewNacosRegistration(RegistrationConfig{Mode: RegistrationModeNacos}, nil)
	if err == nil || registration != nil {
		t.Fatal("nil executor was accepted")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	registration = testRegistrationWithURL(t, server.URL+"/nacos", server.Client(), time.Second)
	if err := registration.Register(canceled); err == nil {
		t.Fatal("canceled registration succeeded")
	}
	if err := registration.Register(t.Context()); err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("unavailable registration error=%v", err)
	}
}

func TestNacosRegistrationRejectsInvalidLifecycleCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registration := testRegistration(t, server, time.Second)
	if registration.Ready() {
		t.Fatal("registration was ready before its initial publish")
	}
	if err := registration.Run(t.Context()); err == nil {
		t.Fatal("lease observation started before registration")
	}
	if err := registration.Run(nil); err == nil {
		t.Fatal("nil lease observation context was accepted")
	}
	if err := registration.Deregister(nil); err == nil {
		t.Fatal("nil deregistration context was accepted")
	}
	if err := registration.Register(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := registration.Register(t.Context()); err == nil {
		t.Fatal("duplicate registration was accepted")
	}
	if err := registration.Deregister(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := registration.Deregister(t.Context()); err != nil {
		t.Fatalf("idempotent deregistration: %v", err)
	}
}

func TestNacosRegistrationRejectsInvalidExactRelease(t *testing.T) {
	config := validRegistrationConfig("http://nacos.test/nacos", time.Second)
	config.CardDigest = "not-a-digest"
	registration, err := NewNacosRegistration(config, http.DefaultClient)
	if err == nil || registration != nil {
		t.Fatal("invalid exact Release target was accepted")
	}
}

func testRegistration(t *testing.T, server *httptest.Server, interval time.Duration) *NacosRegistration {
	t.Helper()
	return testRegistrationWithURL(t, server.URL+"/nacos", server.Client(), interval)
}

func testRegistrationWithURL(t *testing.T, origin string, client HTTPDoer, interval time.Duration) *NacosRegistration {
	t.Helper()
	registration, err := NewNacosRegistration(validRegistrationConfig(origin, interval), client)
	if err != nil {
		t.Fatal(err)
	}
	return registration
}

func validRegistrationConfig(origin string, interval time.Duration) RegistrationConfig {
	return RegistrationConfig{
		Mode: RegistrationModeNacos, AgentID: "runtime-b", InstanceID: "runtime-b-directory",
		AgentCardVersion: "1.0.0", ReleaseID: "rel_runtime_b_1",
		CardDigest:        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092",
		APIOrigin: origin, NamespaceID: "public", GroupName: "NEKIRO", ServiceName: "runtime-b", ClusterName: "DEFAULT",
		PortName: "a2a", AdvertisedIP: "127.0.0.1", AdvertisedPort: 8092, Weight: 1,
		HeartbeatInterval: interval, HeartbeatTimeout: 5 * time.Second, IPDeleteTimeout: 10 * time.Second,
		RequestTimeout: time.Second, AuthMode: NacosAuthNone,
	}
}
