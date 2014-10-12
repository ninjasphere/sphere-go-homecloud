package routes

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-martini/martini"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
)

type RoomRouter struct {
}

func NewRoomRouter() *RoomRouter {
	return &RoomRouter{}
}

func (lr *RoomRouter) Register(r martini.Router) {

	r.Get("", lr.GetAll)
	r.Post("", lr.PostNewRoom)
	r.Get("/:id", lr.GetRoom)
	r.Delete("/:id", lr.DeleteRoom)
	// r.Get("/:id/things", lr.GetThings) Not sure if this was used
	r.Put("/:id/calibrate", lr.PutCalibrateRoom)
	r.Put("/:id/apps/:appName", lr.PutAppRoomMessage)

}

// GetAll retrieves a list of rooms
//
// Response
// [
//    {
//       "id" : "1468fbcd-3ca6-4c6f-a742-ab91221e5462",
//       "things" : [
//          {
//             "type" : "light",
//             "id" : "4b518a5d-f855-4e21-86e0-6e91f6772bea",
//             "device" : "2864dd823a",
//             "name" : "Hue Lamp 2",
//             "location" : "1468fbcd-3ca6-4c6f-a742-ab91221e5462"
//          },
//          {
//             "type" : "light",
//             "id" : "525425b8-7d8e-4da9-9317-a38dd447ece7",
//             "device" : "2df71ceb74",
//             "name" : "Hue Lamp 1",
//             "location" : "1468fbcd-3ca6-4c6f-a742-ab91221e5462"
//          },
//          {
//             "device" : "076ca89411",
//             "id" : "8252f0e2-43d5-4dd2-bf13-834af1b789ca",
//             "type" : "light",
//             "name" : "Hue Lamp",
//             "location" : "1468fbcd-3ca6-4c6f-a742-ab91221e5462"
//          }
//       ],
//       "name" : "Living Room"
//    }
// ]
//
func (lr *RoomRouter) GetAll(w http.ResponseWriter, roomModel *homecloud.RoomModel) {
	rooms, err := roomModel.FetchAll()

	log.Infof(spew.Sprintf("room: %v", rooms))

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve rooms", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(rooms, http.StatusOK, w)
}

// PostNewRoom creates a new room using the name submitted
//
// Request {"name":"Bedroom","type":"bedroom"}
// Response {"name":"Bedroom","type":"bedroom","id":"16c63268-c0e5-48a2-b312-c74c64837802"}
//
func (lr *RoomRouter) PostNewRoom(r *http.Request, w http.ResponseWriter, roomModel *homecloud.RoomModel) {

	// get the request body
	body, err := GetJsonPayload(r)

	if err != nil {
		WriteServerErrorResponse("Unable to parse body", http.StatusInternalServerError, w)
		return
	}

	roomName := body["name"].(string)
	roomType := body["type"].(string)

	room := &model.Room{Name: roomName, Type: roomType}

	err = roomModel.Create(room)

	if err != nil {
		WriteServerErrorResponse("Unable to create room", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(room, http.StatusOK, w)

}

// GetRoom retrieves a room using it's identifier
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

// DeleteRoom removes a room using it's identifier
func (lr *RoomRouter) DeleteRoom(params martini.Params, w http.ResponseWriter, roomModel *homecloud.RoomModel) {

	err := roomModel.Delete(params["id"])

	if err == homecloud.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown room id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to delete room", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK) // TODO: talk to theo about this response.
}

// PutCalibrateRoom enables calibration for a specific room
//
// Request {"id":"1468fbcd-3ca6-4c6f-a742-ab91221e5462","device":"20CD39A0899C","reset":true}
// Response 200
//
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

// PutAppRoomMessage sends a message to the app passing the identifier of the room which it applies too
func (lr *RoomRouter) PutAppRoomMessage(params martini.Params, r *http.Request, w http.ResponseWriter, roomModel *homecloud.RoomModel, conn *ninja.Connection) {

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

	appName := params["appName"]

	if appName == "" {
		WriteServerErrorResponse("appName required", http.StatusBadRequest, w)
		return
	}

	// get the request body
	body, err := GetJsonPayload(r)

	if err != nil {
		WriteServerErrorResponse("unable to parse json body", http.StatusBadRequest, w)
		return
	}

	topic := fmt.Sprintf("$node/%s/app/%s", NodeID, params["appName"])

	// TODO: Need to send message over an RPC thing
	//   req.bus.publish(topic,req.params.id,message,function(err,response){
	conn.SendNotification(topic, room.ID, appName, body)

	w.WriteHeader(http.StatusOK)
}
