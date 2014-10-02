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

	r.Get("/:id", lr.GetRoom)

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
