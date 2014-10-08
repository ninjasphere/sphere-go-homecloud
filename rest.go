package main

import (
	"net/http"

	"github.com/go-martini/martini"
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
		stateManager: state.NewStateManager(conn),
	}
}

func (r *RestServer) Listen() {

	m := martini.Classic()

	m.Map(r.roomModel)
	m.Map(r.thingModel)
	m.Map(r.deviceModel)
	m.Map(r.conn)
	m.Map(r.stateManager)

	location := routes.NewLocationRouter()
	thing := routes.NewThingRouter()
	room := routes.NewRoomRouter()

	m.Group("/rest/v1/locations", location.Register)
	m.Group("/rest/v1/things", thing.Register)
	m.Group("/rest/v1/rooms", room.Register)

	http.ListenAndServe(":8000", m)
}
