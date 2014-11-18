package main

import (
	"net/http"

	"os/exec"
	"syscall"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/cors"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
	"github.com/ninjasphere/sphere-go-homecloud/routes"
	"github.com/ninjasphere/sphere-go-homecloud/state"
)

// RestServer Holds stuff shared by all the rest services
type RestServer struct {
	redisPool *redis.Pool
	conn      *ninja.Connection

	roomModel    *homecloud.RoomModel
	thingModel   *homecloud.ThingModel
	deviceModel  *homecloud.DeviceModel
	siteModel    *homecloud.SiteModel
	stateManager state.StateManager
}

func NewRestServer(conn *ninja.Connection) *RestServer {

	conn, err := ninja.Connect("sphere-go-homecloud-rest")

	if err != nil {
		log.FatalError(err, "Failed to connect to mqtt")
	}

	return &RestServer{
		redisPool:    homecloud.RedisPool,
		conn:         conn,
		roomModel:    homecloud.NewRoomModel(homecloud.RedisPool, conn),
		thingModel:   homecloud.NewThingModel(homecloud.RedisPool, conn),
		deviceModel:  homecloud.NewDeviceModel(homecloud.RedisPool, conn),
		siteModel:    homecloud.NewSiteModel(homecloud.RedisPool, conn),
		stateManager: state.NewStateManager(conn),
	}
}

func (r *RestServer) Listen() {

	m := martini.Classic()

	m.Use(cors.Allow(&cors.Options{
		AllowAllOrigins: true,
	}))

	m.Map(r.roomModel)
	m.Map(r.thingModel)
	m.Map(r.deviceModel)
	m.Map(r.siteModel)
	m.Map(r.conn)
	m.Map(r.stateManager)

	location := routes.NewLocationRouter()
	thing := routes.NewThingRouter()
	room := routes.NewRoomRouter()
	site := routes.NewSiteRouter()

	m.Group("/rest/v1/locations", location.Register)
	m.Group("/rest/v1/things", thing.Register)
	m.Group("/rest/v1/rooms", room.Register)
	m.Group("/rest/v1/sites", site.Register)

	// the following methods are temporary, and will go away at some stage once a real update process is in place
	m.Post("/rest/tmp/apt/update", func() string {
		cmd := exec.Command("/usr/bin/nohup", "/bin/sh", "-c", "apt-get update; apt-get -y dist-upgrade")
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Setpgid = true
		cmd.Start()
		return "OK"
	})

	http.ListenAndServe(":8000", m)
}
