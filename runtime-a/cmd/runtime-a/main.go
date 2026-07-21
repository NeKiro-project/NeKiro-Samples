package main

import (
	"log"
	"net/http"
	"os"

	runtimea "github.com/Nene7ko/NeKiro/agents/runtime-a"
)

func main() {
	config, err := runtimea.LoadConfig(os.LookupEnv)
	if err != nil {
		log.Fatal(err)
	}
	handler, err := runtimea.NewHandler(config, http.DefaultClient)
	if err != nil {
		log.Fatal("runtime-a initialize: ", err)
	}
	if err := http.ListenAndServe(config.ListenAddress, runtimea.NewHTTPHandler(handler)); err != nil {
		log.Fatal("runtime-a serve: ", err)
	}
}
