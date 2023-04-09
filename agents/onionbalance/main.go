package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	local "github.com/bugfest/tor-controller/agents/onionbalance/local"
)

// onionbalance-manager main.
func main() {
	flag.Parse()

	localManager := local.New()

	err := localManager.Run()
	if err != nil {
		log.Fatalf("%v", err)
	}
}
