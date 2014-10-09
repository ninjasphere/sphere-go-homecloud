package homecloud

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-ninja/rpc/json2"
	"github.com/ninjasphere/redigo/redis"
)

var log = logger.GetLogger("HomeCloud")

var conn *ninja.Connection

var thingModel *ThingModel
var deviceModel *DeviceModel
var channelModel *ChannelModel
var roomModel *RoomModel
var driverModel *DriverModel

var locationRegexp = regexp.MustCompile("\\$device\\/([A-F0-9]*)\\/[^\\/]*\\/location")

type incomingLocationUpdate struct {
	Zone *string `json:"zone,omitempty"`
}

type outgoingLocationUpdate struct {
	ID         *string `json:"id"`
	HasChanged bool    `json:"hasChanged"`
}

var RedisPool = &redis.Pool{
	MaxIdle:     3,
	IdleTimeout: 240 * time.Second,
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", ":6379")
		if err != nil {
			return nil, err
		}
		/*if _, err := c.Do("AUTH", password); err != nil {
			c.Close()
			return nil, err
		}*/
		return c, err
	},
	TestOnBorrow: func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	},
}

func Start(c *ninja.Connection) {

	//FIXME
	conn = c

	thingModel = NewThingModel(RedisPool, conn)
	conn.MustExportService(thingModel, "$home/services/ThingModel", &model.ServiceAnnouncement{
		Schema: "/service/thing-model",
	})

	deviceModel = NewDeviceModel(RedisPool, conn)
	conn.MustExportService(deviceModel, "$home/services/DeviceModel", &model.ServiceAnnouncement{
		Schema: "/service/device-model",
	})

	channelModel = NewChannelModel(RedisPool, conn)
	conn.MustExportService(deviceModel, "$home/services/ChannelModel", &model.ServiceAnnouncement{
		Schema: "/service/channel-model",
	})

	roomModel = NewRoomModel(RedisPool, conn)
	conn.MustExportService(roomModel, "$home/services/RoomModel", &model.ServiceAnnouncement{
		Schema: "/service/room-model",
	})

	if config.Bool(false, "clearcloud") {
		log.Infof("Clearing all cloud data in 5 seconds")

		time.Sleep(time.Second * 5)

		thingModel.ClearCloud()
		channelModel.ClearCloud()
		deviceModel.ClearCloud()
		roomModel.ClearCloud()

		log.Infof("All cloud data cleared? Probably.")

		return
	}

	go func() {
		for {
			log.Infof("\n\n\n------ Timed model syncing started (every 30 min) ------ ")

			roomResult := roomModel.sync()
			deviceResult := deviceModel.sync()
			channelResult := channelModel.sync()
			thingResult := thingModel.sync()

			log.Infof("Room sync error: %s", roomResult)
			log.Infof("Device sync error: %s", deviceResult)
			log.Infof("Channel sync error: %s", channelResult)
			log.Infof("Thing sync error: %s", thingResult)

			log.Infof("------ Timed model syncing complete ------\n\n\n")

			time.Sleep(time.Minute * 30)
		}
	}()

	driverModel = NewDriverModel(RedisPool, conn)

	startManagingDrivers()
	startManagingDevices()
	startMonitoringLocations()

	go func() {
		// Give it a chance to sync first...
		time.Sleep(time.Second * 10)
		startDrivers()
	}()

}

func startDrivers() {

	do := func(name string, task string) error {
		return conn.SendNotification("$node/"+config.Serial()+"/module/"+task, name)
	}

	for _, name := range []string{"driver-go-zigbee", "driver-go-sonos", "driver-go-lifx", "driver-go-ble", "driver-go-hue", "driver-go-wemo"} {
		log.Infof("-- (Re)starting '%s'", name)

		err := do(name, "stop")
		if err != nil {
			log.Fatalf("Failed to send %s stop message! %s", name, err)
		}

		time.Sleep(time.Second * 2)

		err = do(name, "start")
		if err != nil {
			log.Fatalf("Failed to send %s start message! %s", name, err)
		}
	}

}

func startDriver(node string, driverID string, config *string) error {

	var rawConfig json.RawMessage
	if config != nil {
		rawConfig = []byte(*config)
	} else {
		rawConfig = []byte("{}")
	}

	client := conn.GetServiceClient(fmt.Sprintf("$node/%s/driver/%s", node, driverID))
	err := client.Call("start", &rawConfig, nil, 10*time.Second)

	if err != nil {
		jsonError, ok := err.(*json2.Error)
		if ok {
			if jsonError.Code == json2.E_INVALID_REQ {

				err := driverModel.DeleteConfig(driverID)
				if err != nil {
					log.Warningf("Driver %s could not parse its config. Also, we couldn't clear it! errors:%s and %s", driverID, jsonError.Message, err)
				} else {
					log.Warningf("Driver %s could not parse its config, so we cleared it from redis. error:%s", driverID, jsonError.Message)
				}

				return startDriver(node, driverID, nil)
			}
		}
	}

	return err
}

func startManagingDrivers() {

	conn.Subscribe("$node/:node/driver/:driver/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		node, driver := values["node"], values["driver"]

		log.Infof("Got driver announcement node:%s driver:%s announcement:%s", node, driver, announcement)

		if announcement == nil {
			log.Warningf("Nil driver announcement from node:%s driver:%s", node, driver)
			return true
		}

		module := &model.Module{}
		err := json.Unmarshal(*announcement, module)

		if announcement == nil {
			log.Warningf("Could not parse announcement from node:%s driver:%s error:%s", node, driver, err)
			return true
		}

		err = driverModel.Create(module)
		if err != nil {
			log.Warningf("Failed to save driver announcement for %s error:%s", driver, err)
		}

		config, err := driverModel.GetConfig(values["driver"])

		if err != nil {
			log.Warningf("Failed to retrieve config for driver %s error:%s", driver, err)
		} else {
			err = startDriver(node, driver, config)
			if err != nil {
				log.Warningf("Failed to start driver: %s error:%s", driver, err)
			}
		}

		return true
	})

	conn.Subscribe("$node/:node/driver/:driver/event/config", func(config *json.RawMessage, values map[string]string) bool {
		log.Infof("Got driver config node:%s driver:%s config:%s", values["node"], values["driver"], *config)

		if config != nil {
			err := driverModel.SetConfig(values["driver"], string(*config))

			if err != nil {
				log.Warningf("Failed to save config for driver: %s error: %s", values["driver"], err)
			}
		} else {
			log.Infof("Nil config recevied from node:%s driver:%s", values["node"], values["driver"])
		}

		return true
	})

}

func startManagingDevices() {

	conn.Subscribe("$device/:id/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		id := values["id"]

		log.Infof("Got device announcement device:%s announcement:%s", id, announcement)

		if announcement == nil {
			log.Warningf("Nil driver announcement from device:%s", id)
			return true
		}

		device := &model.Device{}
		err := json.Unmarshal(*announcement, device)

		if announcement == nil {
			log.Warningf("Could not parse announcement from device:%s error:%s", id, err)
			return true
		}

		err = deviceModel.Create(device)
		if err != nil {
			log.Warningf("Failed to save device announcement for device:%s error:%s", id, err)
		}

		return true
	})

	conn.Subscribe("$device/:device/channel/:channel/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		deviceID, channelID := values["device"], values["channel"]

		log.Infof("Got channel announcement device:%s channel:%s announcement:%s", deviceID, channelID, announcement)

		if announcement == nil {
			log.Warningf("Nil channel announcement from device:%s channel:%s", deviceID, channelID)
			return true
		}

		channel := &model.Channel{}
		err := json.Unmarshal(*announcement, channel)

		if announcement == nil {
			log.Warningf("Could not parse channel announcement from device:%s channel:%s error:%s", deviceID, channelID, err)
			return true
		}

		err = channelModel.Create(deviceID, channel)
		if err != nil {
			log.Warningf("Failed to save channel announcement for device:%s channel:%s error:%s", deviceID, channelID, err)
		}

		return true
	})

}

func startMonitoringLocations() {

	filter, err := mqtt.NewTopicFilter("$device/+/+/location", 0)
	if err != nil {
		log.FatalError(err, "Failed to subscribe to device locations")
	}

	receipt, err := conn.GetMqttClient().StartSubscription(func(_ *mqtt.MqttClient, message mqtt.Message) {

		deviceID := locationRegexp.FindAllStringSubmatch(message.Topic(), -1)[0][1]

		update := &incomingLocationUpdate{}
		err := json.Unmarshal(message.Payload(), update)
		if err != nil {
			log.Errorf("Failed to parse location update %s to %s : %s", message.Payload(), message.Topic(), err)
			return
		}

		thing, err := thingModel.FetchByDeviceId(deviceID)
		if err != nil {
			log.FatalError(err, fmt.Sprintf("Failed to fetch thing by device id %s", deviceID))
		}

		if update.Zone == nil {
			log.Debugf("< Incoming location update: device %s not in a zone", deviceID)
		} else {
			log.Debugf("< Incoming location update: device %s is in zone %s", deviceID, *update.Zone)
		}

		hasChangedZone := true

		if thing == nil {
			log.Debugf("Device %s is not attached to a thing. Ignoring.", deviceID)
		} else {

			if (thing.Location != nil && update.Zone != nil && *thing.Location == *update.Zone) || (thing.Location == nil && update.Zone == nil) {
				// It's already there
				log.Debugf("Thing %s (%s) (Device %s) was already in that zone.", thing.ID, thing.Name, deviceID)
				hasChangedZone = true
			} else {

				log.Debugf("Thing %s (%s) (Device %s) moved from %s to %s", thing.ID, thing.Name, deviceID, thing.Location, update.Zone)

				err = roomModel.MoveThing(thing.Location, update.Zone, thing.ID)

				if err != nil {
					log.FatalError(err, fmt.Sprintf("Failed to update location of thing %s", thing.ID))
				}

				err = thingModel.SetLocation(thing.ID, update.Zone)
				if err != nil {
					log.FatalError(err, fmt.Sprintf("Failed to update location property of thing %s", thing.ID))
				}

				if update.Zone != nil {
					room, err := roomModel.Fetch(*update.Zone)
					if err != nil {
						log.FatalError(err, fmt.Sprintf("Failed to fetch room %s", *update.Zone))
					}

					if room == nil {
						// XXX: TODO: Remove me once the cloud room model is sync'd and locatino service uses it
						log.Infof("Unknown room %s. Advising remote location service to forget it.", *update.Zone)

						pubReceipt := conn.GetMqttClient().Publish(mqtt.QoS(0), "$location/delete", message.Payload())
						<-pubReceipt
					}
				}
			}

			topic := fmt.Sprintf("$device/%s/channel/%s/%s/event/state", deviceID, "location", "location")

			payload, _ := json.Marshal(&outgoingLocationUpdate{
				ID:         update.Zone,
				HasChanged: hasChangedZone,
			})

			pubReceipt := conn.GetMqttClient().Publish(mqtt.QoS(0), topic, payload)
			<-pubReceipt

		}

	}, filter)

	if err != nil {
		log.Fatalf("Failed to subscribe to device locations: %s", err)
	}

	<-receipt
}
