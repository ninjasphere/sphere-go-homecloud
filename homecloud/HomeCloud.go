package homecloud

import (
	"encoding/json"
	"fmt"
	"regexp"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"

	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/redigo/redis"
)

var thingModel *ThingModel
var deviceModel *DeviceModel
var roomModel *RoomModel

var name = "sphere-go-homecloud"

var locationRegexp = regexp.MustCompile("\\$device\\/([A-F0-9]*)\\/[^\\/]*\\/location")

type LocationUpdate struct {
	Zone string `json:"zone"`
}

func Start() {

	log := logger.GetLogger("HomeCloud")

	redisConn, err := redis.Dial("tcp", ":6379")

	if err != nil {
		log.FatalError(err, "Couldn't connect to redit")
	}

	//defer redisConn.Close()

	thingModel = NewThingModel(redisConn)
	deviceModel = NewDeviceModel(redisConn)
	roomModel = NewRoomModel(redisConn)

	conn, err := ninja.Connect(name)

	if err != nil {
		log.FatalError(err, "Failed to connect to mqtt")
	}

	statusJob, err := ninja.CreateStatusJob(conn, name)
	if err != nil {
		log.FatalError(err, "Could not setup status job")
	}

	statusJob.Start()

	filter, err := mqtt.NewTopicFilter("$device/+/+/location", 0)
	if err != nil {
		log.FatalError(err, "Failed to subscribe to device locations")
	}

	receipt, err := conn.GetMqttClient().StartSubscription(func(_ *mqtt.MqttClient, message mqtt.Message) {

		deviceID := locationRegexp.FindAllStringSubmatch(message.Topic(), -1)[0][1]

		update := &LocationUpdate{}
		err := json.Unmarshal(message.Payload(), update)
		if err != nil {
			log.Errorf("Failed to parse location update %s to %s : %s", message.Payload(), message.Topic(), err)
			return
		}

		log.Debugf("< Incoming location update: device %s is in zone %s", deviceID, update.Zone)

		// XXX: TODO: Remove me once the cloud room model is sync'd and locatino service uses it
		room, err := roomModel.Fetch(update.Zone)
		if err != nil {
			log.FatalError(err, fmt.Sprintf("Failed to fetch room %s", update.Zone))
		}

		if room == nil {
			log.Infof("Unknown room %s. Advising remote location service to forget it.", update.Zone)

			pubReceipt := conn.GetMqttClient().Publish(mqtt.QoS(0), "$location/delete", message.Payload())
			<-pubReceipt

			return
		}

		thing, err := thingModel.FetchByDeviceId(deviceID)
		if err != nil {
			log.FatalError(err, fmt.Sprintf("Failed to fetch thing by device id %s", deviceID))
		}

		if thing != nil {
			if *thing.Location == update.Zone {
				// It's already there
				return
			}
			err := roomModel.MoveThing(thing.Location, &update.Zone, thing.ID)

			if err != nil {
				log.FatalError(err, fmt.Sprintf("Failed to update location of thing %s", thing.ID))
			}
		}
	}, filter)

	if err != nil {
		log.Fatalf("Failed to subscribe to device locations: %s", err)
	}

	<-receipt

}
