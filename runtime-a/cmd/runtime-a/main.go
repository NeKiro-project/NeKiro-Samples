package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
	runtimea "github.com/NeKiro-project/NeKiro-Samples/runtime-a"
	agenthost "github.com/NeKiro-project/nekiro-sdk-go/agent/host"
	registrationnacos "github.com/NeKiro-project/nekiro-sdk-go/agent/registration/nacos"
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
	config, err := runtimea.LoadConfig(lookup)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime A configuration", err)
	}
	registrationConfig, err := registrationnacos.LoadConfig(lookup, "RUNTIME_A", config.AgentID, config.InstanceID)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime A registration configuration", err)
	}
	registration, readiness, err := newRuntimeRegistration(registrationConfig)
	if err != nil {
		return agenthost.Wrap(agenthost.StageRegistration, "create Runtime A registration", err)
	}
	handler, err := runtimea.NewHandler(config, http.DefaultClient)
	if err != nil {
		return agenthost.Wrap(agenthost.StageHandler, "create Runtime A handler", err)
	}
	application, err := challengeproof.NewHandler(runtimea.NewHTTPHandlerWithReadiness(handler, readiness), lookup)
	if err != nil {
		return agenthost.Wrap(agenthost.StageHandler, "configure Runtime A endpoint challenge", err)
	}
	shutdownTimeout := 5 * time.Second
	if registrationConfig.RequestTimeout > 0 {
		shutdownTimeout = registrationConfig.RequestTimeout
	}
	runtimeHost, err := agenthost.New(agenthost.Config{
		Address:         config.ListenAddress,
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

func newRuntimeRegistration(config registrationnacos.Config) (agenthost.Registration, runtimea.Readiness, error) {
	if config.Mode == registrationnacos.ModeDisabled {
		return nil, ready(true), nil
	}
	registration, err := registrationnacos.New(config)
	if err != nil {
		return nil, nil, err
	}
	return registration, registration, nil
}
