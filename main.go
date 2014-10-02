package main

import (
	"os"
	"os/signal"

	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
)

var log = logger.GetLogger("HomeCloud")

func main() {

	homecloud.Start()

	log.Infof("hello")

	restServer := NewRestServer() // TODO: Should we reuse the persistance layer from homecloud?

	go restServer.Listen()

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Infof("Got signal: %v", x)

}
