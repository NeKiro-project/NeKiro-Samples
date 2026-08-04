package main

import (
	"log"
	"net/http"
	"os"

	"github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
	runtimeb "github.com/NeKiro-project/NeKiro-Samples/runtime-b"
	"github.com/NeKiro-project/nekiro-sdk-go/agent/routerauth"
)

func main() {
	address, err := runtimeb.ListenAddressFromEnvironment(os.LookupEnv)
	if err != nil {
		log.Fatal(err)
	}
	authenticationConfig, err := routerauth.LoadConfig(os.LookupEnv)
	if err != nil {
		log.Fatal("runtime-b authentication config: ", err)
	}
	config, err := runtimeb.LoadConfig(os.LookupEnv)
	if err != nil {
		log.Fatal(err)
	}
	handler, err := runtimeb.NewConfiguredHandler(config, http.DefaultClient)
	if err != nil {
		log.Fatal("runtime-b initialize: ", err)
	}
	execution, err := runtimeb.NewHTTPHandlerWithAuth(handler, authenticationConfig)
	if err != nil {
		log.Fatal("runtime-b authentication: ", err)
	}
	application, err := challengeproof.NewHandler(execution, os.LookupEnv)
	if err != nil {
		log.Fatal("runtime-b challenge proof: ", err)
	}
	if err := http.ListenAndServe(address, application); err != nil {
		log.Fatal(err)
	}
}
