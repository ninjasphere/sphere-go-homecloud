package main

import (
	"net/http"

	"github.com/go-martini/martini"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
	"github.com/ninjasphere/sphere-go-homecloud/routes"
)

// RestServer Holds stuff shared by all the rest services
type RestServer struct {
	redisPool *redis.Pool
	conn      *ninja.Connection

	roomModel   *homecloud.RoomModel
	thingModel  *homecloud.ThingModel
	deviceModel *homecloud.DeviceModel
}

func NewRestServer(conn *ninja.Connection) *RestServer {

	conn, err := ninja.Connect("sphere-go-homecloud-rest")

	if err != nil {
		log.FatalError(err, "Failed to connect to mqtt")
	}

	return &RestServer{
		redisPool:   homecloud.RedisPool,
		conn:        conn,
		roomModel:   homecloud.NewRoomModel(homecloud.RedisPool, conn),
		thingModel:  homecloud.NewThingModel(homecloud.RedisPool, conn),
		deviceModel: homecloud.NewDeviceModel(homecloud.RedisPool, conn),
	}
}

func (r *RestServer) Listen() {

	m := martini.Classic()

	m.Map(r.roomModel)
	m.Map(r.thingModel)
	m.Map(r.deviceModel)
	m.Map(r.conn)

	location := routes.NewLocationRouter()
	thing := routes.NewThingRouter()
	room := routes.NewRoomRouter()

	m.Group("/rest/v1/location", location.Register)
	m.Group("/rest/v1/thing", thing.Register)
	m.Group("/rest/v1/room", room.Register)

	http.ListenAndServe(":8000", m)
}

func (r *RestServer) getStuff() string {
	return "Hello world!"
}
