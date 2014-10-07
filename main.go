package main

import (
	"os"
	"os/signal"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
)

var log = logger.GetLogger("HomeCloud")

func main() {

	conn, err := ninja.Connect("sphere-go-homecloud")
	if err != nil {
		log.Fatalf("Failed to connect to sphere: %s", err)
	}

	homecloud.Start(conn)

	log.Infof("hello")

	restServer := NewRestServer(conn) // TODO: Should we reuse the persistance layer from homecloud?

	go restServer.Listen()

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Infof("Got signal: %v", x)

}
