package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-martini/martini"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

type ThingRouter struct {
}

func NewThingRouter() *ThingRouter {
	return &ThingRouter{}
}

func (lr *ThingRouter) Register(r martini.Router) {

	r.Get("", lr.GetAll)
	r.Get("/:id", lr.GetThing)
	r.Put("/:id", lr.PutThing)
	r.Put("/:id/location", lr.PutThingLocation)
	r.Delete("/:id", lr.DeleteThing)

}

// GetAll retrieves a list of all things
func (lr *ThingRouter) GetAll(r *http.Request, w http.ResponseWriter, thingModel *models.ThingModel, conn redis.Conn) {
	// if type is specified as a query param
	qs := r.URL.Query()

	var err error
	var things *[]*model.Thing

	if qs.Get("type") != "" {
		things, err = thingModel.FetchByType(qs.Get("type"), conn)
	} else {
		things, err = thingModel.FetchAll(conn)
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve things", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(things, http.StatusOK, w)
}

// GetThing retrieves a thing using it's identifier
func (lr *ThingRouter) GetThing(params martini.Params, w http.ResponseWriter, thingModel *models.ThingModel, conn redis.Conn) {

	thing, err := thingModel.Fetch(params["id"], conn)

	log.Infof(spew.Sprintf("thing: %v", thing))

	if err == models.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown thing id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve thing", http.StatusInternalServerError, w)
		return
	}

	WriteServerResponse(thing, http.StatusOK, w)
}

// GetAll updates a thing using it's identifier, with the JSON payload containing name and type
func (lr *ThingRouter) PutThing(params martini.Params, r *http.Request, w http.ResponseWriter, thingModel *models.ThingModel, conn redis.Conn) {

	var thing *model.Thing

	err := json.NewDecoder(r.Body).Decode(&thing)

	if err != nil {
		WriteServerErrorResponse("Unable to parse body", http.StatusInternalServerError, w)
		return
	}

	err = thingModel.Update(params["id"], thing, conn)

	if err != nil {
		WriteServerErrorResponse("Unable to update thing", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// PutThingLocation assigns or clears the location for a thing, this is currently a room identifier sent in the payload
func (lr *ThingRouter) PutThingLocation(params martini.Params, r *http.Request, w http.ResponseWriter, thingModel *models.ThingModel, conn redis.Conn) {

	thing, err := thingModel.Fetch(params["id"], conn)

	log.Infof(spew.Sprintf("thing: %v", thing))

	if err == models.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown thing id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to retrieve thing", http.StatusInternalServerError, w)
		return
	}

	// get the request body
	body, err := GetJsonPayload(r)

	if err != nil {
		WriteServerErrorResponse("Unable to parse body", http.StatusInternalServerError, w)
		return
	}

	roomID := body["id"].(string)

	// not a big fan of this magic
	if roomID == "" {
		err = thingModel.SetLocation(params["id"], nil, conn)
	} else {
		err = thingModel.SetLocation(params["id"], &roomID, conn)
	}

	if err != nil {
		WriteServerErrorResponse("Unable to save thing location", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteThing removes a thing using it's identifier
func (lr *ThingRouter) DeleteThing(params martini.Params, w http.ResponseWriter, thingModel *models.ThingModel, conn redis.Conn) {

	err := thingModel.Delete(params["id"], true, conn)

	if err == models.RecordNotFound {
		WriteServerErrorResponse(fmt.Sprintf("Unknown thing id: %s", params["id"]), http.StatusNotFound, w)
		return
	}

	if err != nil {
		WriteServerErrorResponse("Unable to delete thing", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusOK) // TODO: talk to theo about this response.
}
