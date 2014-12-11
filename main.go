package main

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/inject"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
	"github.com/ninjasphere/sphere-go-homecloud/models"
	"github.com/ninjasphere/sphere-go-homecloud/rest"
	"github.com/ninjasphere/sphere-go-homecloud/state"
)

var log = logger.GetLogger("HomeCloud")

type postConstructable interface {
	PostConstruct() error
}

func main() {

	log.Infof("Welcome home, Ninja.")

	conn, err := ninja.Connect("sphere-go-homecloud")
	if err != nil {
		log.Fatalf("Failed to connect to sphere: %s", err)
	}

	pool := &redis.Pool{
		MaxIdle:     config.MustInt("homecloud.redis.maxIdle"),
		IdleTimeout: config.MustDuration("homecloud.redis.idleTimeout"),
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf(":%d", config.MustInt("homecloud.redis.port")))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	injectables := []interface{}{}

	injectables = append(injectables, pool, conn)
	injectables = append(injectables, &homecloud.HomeCloud{})
	injectables = append(injectables, state.NewStateManager())
	injectables = append(injectables, &rest.RestServer{}, &homecloud.WebsocketServer{})
	injectables = append(injectables, models.GetInjectables()...)

	err = inject.Populate(injectables...)

	if err != nil {
		log.Fatalf("Failed to construct the object graph: %s", err)
	}

	for _, node := range injectables {
		if n, ok := node.(postConstructable); ok {
			go func(c postConstructable) {
				if err := c.PostConstruct(); err != nil {
					log.Fatalf("Failed PostConstruct on object %s: %s", reflect.TypeOf(c).String(), err)
				}
			}(n)
		}
	}
	/*go NewWebsocketServer(conn)

	homecloud.Start(conn)

	restServer := NewRestServer(conn) // TODO: Should we reuse the persistance layer from homecloud?

	go restServer.Listen()*/

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	log.Infof("Got signal: %v", <-sig)

}
