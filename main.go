package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
)

func main() {

	homecloud.StartHomeCloud()

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Println("Got signal:", x)

}
