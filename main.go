// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/ninjasphere/go-homecloud"
)

func errHndlr(err error) {
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func main() {

	homecloud.StartHomeCloud()

	blah := make(chan os.Signal, 1)
	signal.Notify(blah, os.Interrupt, os.Kill)

	// Block until a signal is received.
	x := <-blah
	log.Println("Got signal:", x)
	/*redisAddress := ":6379"
	maxConnections := 10

	redisPool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", redisAddress)

		if err != nil {
			return nil, err
		}

		return c, err
	}, maxConnections)

	c := redisPool.Get()

	c, err := redis.Dial("tcp", ":6379")

	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	err = c.Send("SMEMBERS", "device")
	if err != nil {
		fmt.Println(err)
	}
	c.Flush()
	// both give the same return value!?!?
	// reply, err := c.Receive()
	spew.Dump(c.Receive())
	reply, err := redis.MultiBulk(c.Receive())*
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%#v\n", reply)*/

	/*item, err := redis.Values(c.Do("HGETALL", "device:13294d4619"))

	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(item)

	device := &model.Device{}

	if err := redis.ScanStruct(item, device); err != nil {
		log.Fatal(err)
	}

	spew.Dump(item, device, err)*/

}
