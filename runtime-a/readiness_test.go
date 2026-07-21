package runtimea

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRuntimeAReadinessDoesNotInvokeNestedWork(t *testing.T) {
	config, err := LoadConfig(lookupEnvironment(validEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	invoker := &recordingInvoker{}
	handler, err := newHandlerWithInvoker(config, invoker)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	NewHTTPHandler(handler).ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(invoker.calls) != 0 {
		t.Fatalf("readiness status=%d nested calls=%d", response.Code, len(invoker.calls))
	}
}
