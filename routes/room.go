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
