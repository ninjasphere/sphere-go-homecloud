package main

import (
	"github.com/go-martini/martini"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
	"github.com/ninjasphere/sphere-go-homecloud/routes"
)

// RestServer Holds stuff shared by all the rest services
type RestServer struct {
	redisPool *redis.Pool

	roomModel  *homecloud.RoomModel
	thingModel *homecloud.ThingModel
}

func NewRestServer(conn *ninja.Connection) *RestServer {

	return &RestServer{
		redisPool:  homecloud.RedisPool,
		roomModel:  homecloud.NewRoomModel(homecloud.RedisPool, conn),
		thingModel: homecloud.NewThingModel(homecloud.RedisPool, conn),
	}
}

func (r *RestServer) Listen() {

	m := martini.Classic()

	m.Map(r.roomModel)
	m.Map(r.thingModel)

	location := routes.NewLocationRouter()
	thing := routes.NewThingRouter()
	room := routes.NewRoomRouter()
	app := routes.NewAppRouter()

	m.Group("/rest/v1/locations", location.Register)
	m.Group("/rest/v1/things", thing.Register)
	m.Group("/rest/v1/rooms", room.Register)
	m.Group("/rest/v1/apps", app.Register)

	m.Run()
}

func (r *RestServer) getStuff() string {
	return "Hello world!"
}
