package routes

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
)

var wrapped = true
var log = logger.GetLogger("HomeCloud.Router")

// doesn't change so we cache it.
var NodeID = config.Serial()

// ResponseWrapper used to wrap responses from the API
type ResponseWrapper struct {
	Type string      `json:"type,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// WriteServerErrorResponse Builds the wrapped error for the client
func WriteServerErrorResponse(msg string, code int, w http.ResponseWriter) {

	body, err := json.Marshal(&ResponseWrapper{Type: "error", Data: msg})

	if err != nil {
		log.Errorf("Unable to serialise response: %s", err)
	}

	http.Error(w, string(body), code)
}

// WriteServerResponse Builds the wrapped object for the client
func WriteServerResponse(data interface{}, code int, w http.ResponseWriter) {

	var resp interface{}

	if wrapped {
		resp = &ResponseWrapper{Type: "object", Data: data}

	} else {
		resp = data
	}

	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("Unable to serialise response: %s", err)
	}
}

func GetJsonPayload(r *http.Request) (map[string]interface{}, error) {

	var f interface{}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &f)

	if err != nil {
		return nil, err
	}

	return f.(map[string]interface{}), err
}
