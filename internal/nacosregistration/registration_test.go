package nacosregistration

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegistrationUsesCoreLeaseAndDeregisters(t *testing.T) {
	var deletes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete {
			deletes.Add(1)
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registration, err := New(validConfig(server.URL+"/nacos"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if registration.Ready() {
		t.Fatal("registration was ready before publish")
	}
	if err := registration.Register(t.Context()); err != nil || !registration.Ready() {
		t.Fatalf("register ready=%v error=%v", registration.Ready(), err)
	}
	if err := registration.Deregister(t.Context()); err != nil || registration.Ready() {
		t.Fatalf("deregister ready=%v error=%v", registration.Ready(), err)
	}
	if err := registration.Deregister(t.Context()); err != nil || deletes.Load() != 1 {
		t.Fatalf("idempotent deregister deletes=%d error=%v", deletes.Load(), err)
	}
}

func TestRegistrationLeaseFailureIsTerminal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPut {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registration, err := New(validConfig(server.URL+"/nacos"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := registration.Register(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := registration.Run(t.Context()); err == nil || registration.Ready() {
		t.Fatalf("terminal lease error=%v ready=%v", err, registration.Ready())
	}
	if err := registration.Deregister(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func validConfig(origin string) Config {
	return Config{
		Mode: ModeNacos, AgentID: "runtime-b", InstanceID: "runtime-b-primary", AgentCardVersion: "1.0.0", ReleaseID: "rel_runtime_b_1",
		CardDigest:        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092", APIOrigin: origin,
		NamespaceID: "public", GroupName: "NEKIRO", ServiceName: "runtime-b", ClusterName: "DEFAULT", PortName: "a2a",
		AdvertisedIP: "127.0.0.1", AdvertisedPort: 8092, Weight: 1, HeartbeatInterval: time.Second,
		HeartbeatTimeout: 5 * time.Second, IPDeleteTimeout: 10 * time.Second, RequestTimeout: time.Second, AuthMode: AuthNone,
	}
}
