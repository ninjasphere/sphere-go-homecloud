package routes

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-martini/martini"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
)

type RoomRouter struct {
}

func NewRoomRouter() *RoomRouter {
	return &RoomRouter{}
}

func (lr *RoomRouter) Register(r martini.Router) {

	r.Get("/", lr.GetAll)
	r.Get("/:id", lr.GetRoom)
	r.Delete("/:id", lr.DeleteRoom)
	// r.Get("/:id/things", lr.GetThings) Not sure if this was used
	r.Put("/:id/calibrate", lr.PutCalibrateRoom)
	r.Put("/:id/apps/:appName", lr.PutAppRoomMessage)

}

func (lr *RoomRouter) GetAll(w http.ResponseWriter, roomModel *homecloud.RoomModel) {
	rooms, err := roomModel.FetchAll()

	log.Infof(spew.Sprintf("room: %v", rooms))

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve room", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(rooms, http.StatusOK, w)
}

func (lr *RoomRouter) GetRoom(params martini.Params, w http.ResponseWriter, roomModel *homecloud.RoomModel) {

	room, err := roomModel.Fetch(params["id"])

	log.Infof(spew.Sprintf("room: %v", room))

	if err == homecloud.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown room id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve room", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(room, http.StatusOK, w)
}

func (lr *RoomRouter) DeleteRoom(params martini.Params, w http.ResponseWriter, roomModel *homecloud.RoomModel) {

	err := roomModel.Delete(params["id"])

	if err == homecloud.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown room id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve room", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK) // TODO: talk to theo about this response.
}

func (lr *RoomRouter) PutCalibrateRoom(params martini.Params, w http.ResponseWriter, roomModel *homecloud.RoomModel) {

	room, err := roomModel.Fetch(params["id"])

	log.Infof(spew.Sprintf("room: %v", room))

	if err == homecloud.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown room id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve room", http.StatusInternalServerError, w)
		return
	}

	// TODO: Need to send message over an RPC thing
	// LocationManager.calibrate(req.params.id, req.body.device, req.body.reset||false, function(err) {

	w.WriteHeader(http.StatusOK)
}

func (lr *RoomRouter) PutAppRoomMessage(params martini.Params, w http.ResponseWriter, roomModel *homecloud.RoomModel) {

	room, err := roomModel.Fetch(params["id"])

	log.Infof(spew.Sprintf("room: %v", room))

	if err == homecloud.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown room id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve room", http.StatusInternalServerError, w)
		return
	}

	// TODO: Need to send message over an RPC thing
	//   req.bus.publish(topic,req.params.id,message,function(err,response){

	w.WriteHeader(http.StatusOK)
}
