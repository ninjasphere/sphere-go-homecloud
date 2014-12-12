package rest

import (
	"fmt"
	"net/http"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/cors"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/models"
	"github.com/ninjasphere/sphere-go-homecloud/state"
)

// RestServer Holds stuff shared by all the rest services
type RestServer struct {
	RedisPool    *redis.Pool         `inject:""`
	Conn         *ninja.Connection   `inject:""`
	RoomModel    *models.RoomModel   `inject:""`
	ThingModel   *models.ThingModel  `inject:""`
	DeviceModel  *models.DeviceModel `inject:""`
	SiteModel    *models.SiteModel   `inject:""`
	StateManager state.StateManager  `inject:""`
	log          *logger.Logger
}

func (r *RestServer) PostConstruct() error {
	r.log = logger.GetLogger("RestServer")
	return r.Listen()
}

func (r *RestServer) Listen() error {

	m := martini.Classic()

	m.Use(cors.Allow(&cors.Options{
		AllowAllOrigins: true,
	}))

	m.Map(r.RoomModel)
	m.Map(r.ThingModel)
	m.Map(r.DeviceModel)
	m.Map(r.SiteModel)
	m.Map(r.Conn)
	m.Map(r.StateManager)

	location := NewLocationRouter()
	thing := NewThingRouter()
	room := NewRoomRouter()
	site := NewSiteRouter()

	m.Group("/rest/v1/locations", location.Register)
	m.Group("/rest/v1/things", thing.Register)
	m.Group("/rest/v1/rooms", room.Register)
	m.Group("/rest/v1/sites", site.Register)

	listenAddress := fmt.Sprintf(":%d", config.MustInt("homecloud.rest.port"))

	r.log.Infof("Listening at %s", listenAddress)

	return http.ListenAndServe(listenAddress, m)
}
