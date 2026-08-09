package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
	"github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"
	runtimea "github.com/NeKiro-project/NeKiro-Samples/runtime-a"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := runtimea.LoadConfig(os.LookupEnv)
	if err != nil {
		return err
	}
	registrationConfig, err := nacosregistration.Load(os.LookupEnv, "RUNTIME_A", config.InstanceID)
	if err != nil {
		return err
	}
	var registration *nacosregistration.Registration
	var readiness runtimea.Readiness = ready(true)
	if registrationConfig.Mode == nacosregistration.ModeNacos {
		registration, err = nacosregistration.New(registrationConfig, newNacosHTTPClient(registrationConfig.RequestTimeout))
		if err != nil {
			return fmt.Errorf("runtime-a Nacos registration config: %w", err)
		}
		readiness = registration
	}
	handler, err := runtimea.NewHandler(config, http.DefaultClient)
	if err != nil {
		return fmt.Errorf("runtime-a initialize: %w", err)
	}
	application, err := challengeproof.NewHandler(runtimea.NewHTTPHandlerWithReadiness(handler, readiness), os.LookupEnv)
	if err != nil {
		return fmt.Errorf("runtime-a challenge proof: %w", err)
	}
	if registration != nil {
		if err := registration.Register(context.Background()); err != nil {
			return err
		}
	}
	server := &http.Server{Addr: config.ListenAddress, Handler: application}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("runtime-a serve: %w", err)
			return
		}
		serverErrors <- nil
	}()
	var registrationErrors chan error
	if registration != nil {
		registrationErrors = make(chan error, 1)
		go func() { registrationErrors <- registration.Run(ctx) }()
	}
	var runErr error
	registrationStopped := registration == nil
	select {
	case <-ctx.Done():
	case runErr = <-serverErrors:
	case runErr = <-registrationErrors:
		registrationStopped = true
	}
	stop()
	shutdownTimeout := 5 * time.Second
	if registrationConfig.RequestTimeout > 0 {
		shutdownTimeout = registrationConfig.RequestTimeout
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownContext)
	if !registrationStopped {
		select {
		case registrationErr := <-registrationErrors:
			runErr = errors.Join(runErr, registrationErr)
		case <-shutdownContext.Done():
			runErr = errors.Join(runErr, errors.New("Runtime A Nacos heartbeat did not stop before shutdown"))
		}
	}
	var deregisterErr error
	if registration != nil {
		deregisterErr = registration.Deregister(shutdownContext)
	}
	return errors.Join(runErr, shutdownErr, deregisterErr)
}

type ready bool

func (value ready) Ready() bool { return bool(value) }

func newNacosHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableKeepAlives = true
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("Nacos redirects are disabled")
		},
	}
}
