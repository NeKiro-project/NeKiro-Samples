package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
	runtimeb "github.com/NeKiro-project/NeKiro-Samples/runtime-b"
	agenthost "github.com/NeKiro-project/nekiro-sdk-go/agent/host"
	registrationnacos "github.com/NeKiro-project/nekiro-sdk-go/agent/registration/nacos"
	"github.com/NeKiro-project/nekiro-sdk-go/agent/routerauth"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	return runWithLookup(os.LookupEnv)
}

func runWithLookup(lookup func(string) (string, bool)) error {
	address, err := runtimeb.ListenAddressFromEnvironment(lookup)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime B listen address", err)
	}
	authenticationConfig, err := routerauth.LoadConfig(lookup)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime B authentication configuration", err)
	}
	config, err := runtimeb.LoadConfig(lookup)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime B configuration", err)
	}
	registrationConfig, err := registrationnacos.LoadConfig(lookup, "RUNTIME_B", config.AgentID, config.InstanceID)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime B registration configuration", err)
	}
	registration, readiness, err := newRuntimeRegistration(registrationConfig)
	if err != nil {
		return agenthost.Wrap(agenthost.StageRegistration, "create Runtime B registration", err)
	}
	handler, err := runtimeb.NewConfiguredHandler(config, http.DefaultClient)
	if err != nil {
		return agenthost.Wrap(agenthost.StageHandler, "create Runtime B handler", err)
	}
	execution, err := runtimeb.NewHTTPHandlerWithAuthAndReadiness(handler, authenticationConfig, readiness)
	if err != nil {
		return agenthost.Wrap(agenthost.StageHandler, "configure Runtime B authentication", err)
	}
	application, err := challengeproof.NewHandler(execution, lookup)
	if err != nil {
		return agenthost.Wrap(agenthost.StageHandler, "configure Runtime B endpoint challenge", err)
	}
	shutdownTimeout := 5 * time.Second
	if registrationConfig.RequestTimeout > 0 {
		shutdownTimeout = registrationConfig.RequestTimeout
	}
	runtimeHost, err := agenthost.New(agenthost.Config{
		Address:         address,
		Handler:         application,
		Registration:    registration,
		ShutdownTimeout: shutdownTimeout,
		Signals:         []os.Signal{os.Interrupt, syscall.SIGTERM},
	})
	if err != nil {
		return err
	}
	return runtimeHost.Run(context.Background())
}

type ready bool

func (value ready) Ready() bool { return bool(value) }

func newRuntimeRegistration(config registrationnacos.Config) (agenthost.Registration, runtimeb.Readiness, error) {
	if config.Mode == registrationnacos.ModeDisabled {
		return nil, ready(true), nil
	}
	registration, err := registrationnacos.New(config)
	if err != nil {
		return nil, nil, err
	}
	return registration, registration, nil
}
