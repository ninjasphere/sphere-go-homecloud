package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/rpc"
	"github.com/ninjasphere/go-ninja/support"
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

const shortForm = "2006-Jan-02"

var epoch, _ = time.Parse(shortForm, "2014-Dec-01")

func waitForNTP() {
	for {
		if time.Now().After(epoch) {
			break
		}

		time.Sleep(time.Second * 2)
	}
}

func main() {

	log.Infof("Welcome home, Ninja.")

	if config.Bool(true, "homecloud.waitForNTP") {
		waitForNTP()
	}

	// The MQTT Connection
	conn, err := ninja.Connect("sphere-go-homecloud")
	if err != nil {
		log.Fatalf("Failed to connect to sphere: %s", err)
	}

	// An MQTT Connection used for outbound syncing connections
	syncConn := &models.SyncConnection{}
	syncConn.Conn, err = ninja.Connect("sphere-go-homecloud.sync")
	if err != nil {
		log.Fatalf("Failed to connect to sphere (sync): %s", err)
	}

	// Our redis pool
	pool := &redis.Pool{
		MaxIdle:     config.MustInt("homecloud.redis.maxIdle"),
		MaxActive:   config.Int(10, "homecloud.redis.maxActive"),
		IdleTimeout: config.MustDuration("homecloud.redis.idleTimeout"),
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", config.String("", "homecloud.redis.host"), config.MustInt("homecloud.redis.port")))
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

	// Not pretty.
	rpc.RedisPool = pool

	// Wait until we connect to redis successfully.
	for {
		c := pool.Get()

		if c.Err() == nil {
			c.Close()
			break
		}
		log.Warningf("Failed to connect to redis: %s", c.Err())
		time.Sleep(time.Second)
	}

	// Build the object graph using dependency injection
	injectables := []interface{}{}

	injectables = append(injectables, pool, conn, syncConn)
	injectables = append(injectables, &homecloud.HomeCloud{}, &homecloud.TimeSeriesManager{}, &homecloud.DeviceManager{}, &homecloud.ModuleManager{})
	injectables = append(injectables, state.NewStateManager())
	injectables = append(injectables, &rest.RestServer{})
	injectables = append(injectables, models.GetInjectables()...)

	err = inject.Populate(injectables...)

	if err != nil {
		log.Fatalf("Failed to construct the object graph: %s", err)
	}

	// Run PostConstruct on any objects that have it
	for _, node := range injectables {
		if n, ok := node.(postConstructable); ok {
			go func(c postConstructable) {
				if err := c.PostConstruct(); err != nil {
					log.Fatalf("Failed PostConstruct on object %s: %s", reflect.TypeOf(c).String(), err)
				}
			}(n)
		}
	}

	support.WaitUntilSignal()
	// So long, and thanks for all the fish.
}
