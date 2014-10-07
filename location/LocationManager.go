package homecloud

import (
	"encoding/json"

	"github.com/garyburd/redigo/redis"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
)

type LocationManager interface {
	Calibrate(roomID, deviceID string, reset bool) error
}

type CalibrationDevice struct {
	Device string
	Name   *string
	Rssi   int
}

type NinjaLocationManager struct {
	redisPool *redis.Pool

	log               *logger.Logger
	conn              *ninja.Connection
	calibrationDevice string
	calibrationZone   string
	isCalibrating     bool
}

func NewLocationManager(redisPool *redis.Pool) LocationManager {

	log := logger.GetLogger("sphere-go-homecloud-location")

	conn, err := ninja.Connect("sphere-go-homecloud-location")

	if err != nil {
		log.FatalError(err, "Failed to connect to mqtt")
	}

	locationManager := &NinjaLocationManager{
		redisPool: redisPool,
		log:       log,
		conn:      conn,
	}

	conn.MustExportService(locationManager, "$home/LocationManager", &model.ServiceAnnouncement{
		Schema: "/service/location",
	})

	conn.Subscribe("$device/:deviceId/:channel/rssi", locationManager.handRSSI)

	conn.Subscribe("$location/calibration/progress", locationManager.handleCalibrationProgress)

	return locationManager
}

func (n *NinjaLocationManager) Calibrate(roomID, deviceID string, reset bool) error {

	if reset {

		payload, _ := json.Marshal(&map[string]string{
			"zone": roomID,
		})

		// delete the room calibration information
		n.conn.GetMqttClient().Publish(0, "$location/delete", payload)
	}

	return nil
}

// func (n *NinjaLocationManager) GetCalibrationDevice() (<-chan CalibrationDevice, error) {

// 	//$device/:deviceId/:channel/rssi

// }

func (n *NinjaLocationManager) handRSSI(announcement *json.RawMessage, values map[string]string) bool {
	return true
}

func (n *NinjaLocationManager) handleCalibrationProgress(announcement *json.RawMessage, values map[string]string) bool {
	return true
}
