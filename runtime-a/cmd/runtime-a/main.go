package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
	"github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"
	runtimea "github.com/NeKiro-project/NeKiro-Samples/runtime-a"
	agenthost "github.com/NeKiro-project/nekiro-sdk-go/agent/host"
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
	registrationConfig, err := nacosregistration.Load(lookup, "RUNTIME_A", config.AgentID, config.InstanceID)
	if err != nil {
		return agenthost.Wrap(agenthost.StageConfig, "load Runtime A registration configuration", err)
	}
	var registration agenthost.Registration
	var readiness runtimea.Readiness = ready(true)
	if registrationConfig.Mode == nacosregistration.ModeNacos {
		registrationClient, clientErr := nacosregistration.NewHTTPClient(registrationConfig)
		if clientErr != nil {
			return agenthost.Wrap(agenthost.StageRegistration, "create Runtime A Nacos transport", clientErr)
		}
		runtimeRegistration, err := nacosregistration.New(registrationConfig, registrationClient)
		if err != nil {
			return agenthost.Wrap(agenthost.StageRegistration, "create Runtime A registration", err)
		}
		registration = runtimeRegistration
		readiness = runtimeRegistration
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
