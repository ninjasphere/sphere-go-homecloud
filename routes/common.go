package routes

import (
	"encoding/json"
	"net/http"

	"github.com/ninjasphere/go-ninja/logger"
)

var log = logger.GetLogger("HomeCloud.Router")

// ResponseWrapper used to wrap responses from the API
type ResponseWrapper struct {
	Type string
	Data interface{}
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
	resp := &ResponseWrapper{Type: "object", Data: data}
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("Unable to serialise response: %s", err)
	}
}

func GetJsonPayload(r *http.Request) (map[string]interface{}, error) {

	var f interface{}

	err := json.NewDecoder(r.Body).Decode(&f)

	return f.(map[string]interface{}), err
}
