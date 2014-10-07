package routes

import (
	"encoding/json"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-martini/martini"
	"github.com/ninjasphere/sphere-go-homecloud/homecloud"
)

type LocationRouter struct {
}

func NewLocationRouter() *LocationRouter {
	return &LocationRouter{}
}

func (lr *LocationRouter) Register(r martini.Router) {

	r.Get("/calibration/scores", lr.GetCalibrateScores)
	r.Get("/calibration/device", lr.GetCalibrateDevice)
	r.Get("/calibration/progress", lr.GetCalibrationProgress)

	r.Post("/thing", lr.PostCreateThing)

}

// GetCalibrateScores This retrieves the calibration scores for a room
//
// NOTE: Currently returns 504 Gateway Time-out on my sphere
//
func (lr *LocationRouter) GetCalibrateScores(r *http.Request, w http.ResponseWriter) {

	// req.bus.callMethod('$ninja/services/rpc/' + rpcMethod.replace('.', '/'), rpcMethod, args, function(err, result) {
	//   if (err) {
	//     return res.json(500, {
	//       type: 'error',
	//       error: { code: 500, type: 'internal_error', message: 'An internal server error occured.'}
	//     });
	//   } else {
	//     return res.json({
	//       type: 'simple_calibration_scores',
	//       data: result || {}
	//     });
	//   }
	// });

	resp := &ResponseWrapper{Type: "simple_calibration_scores", Data: struct{}{}}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("Unable to serialise response: %s", err)
	}

}

// GetCalibrateDevice
//
// NOTE: This currently gets hit a ton by the web ui when you click the Calibrate button on the site.
//
func (lr *LocationRouter) GetCalibrateDevice(r *http.Request, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK) // TODO: work out how this will be done.
}

func (lr *LocationRouter) GetCalibrationProgress(r *http.Request, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK) // TODO: work out how this will be done.
}

// PostCreateThing this will create a device and a thing from bluetooth devices discovered.
//
// Sends application/json with content {"deviceId":"F0E4FFFFFFFF","thingName":"Bob","thingType":"person"}
//
// Responds with {"success":true,"id":"7efa45b0-2108-464d-83c6-8af1785bc9ea"}
//
func (lr *LocationRouter) PostCreateThing(r *http.Request, w http.ResponseWriter, thingModel *homecloud.ThingModel, deviceModel *homecloud.DeviceModel) {

	// get the request body
	body, err := GetJsonPayload(r)

	if err != nil {
		WriteServerErrorResponse("Unable to parse body", http.StatusInternalServerError, w)
		return
	}

	log.Infof(spew.Sprintf("posted body : %v", body))

	//	deviceID := body["deviceId"].(string)

	w.WriteHeader(http.StatusOK) // TODO: work out how this will be done.

}
