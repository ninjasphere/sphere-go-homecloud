package homecloud

import (
	"regexp"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

var log = logger.GetLogger("HomeCloud")

type HomeCloud struct {
	Conn         *ninja.Connection    `inject:""`
	Pool         *redis.Pool          `inject:""`
	ThingModel   *models.ThingModel   `inject:""`
	DeviceModel  *models.DeviceModel  `inject:""`
	ChannelModel *models.ChannelModel `inject:""`
	RoomModel    *models.RoomModel    `inject:""`
	DriverModel  *models.DriverModel  `inject:""`
	AppModel     *models.AppModel     `inject:""`
	SiteModel    *models.SiteModel    `inject:""`
	log          *logger.Logger
}

func (c *HomeCloud) PostConstruct() error {

	c.log = logger.GetLogger("HomeCloud")

	// We wait for at least one sync to happen, or fail
	<-c.StartSyncing(config.MustDuration("homecloud.sync.interval"))

	return nil
}

// if config.Bool(false, "clearcloud") {

func (c *HomeCloud) ClearCloud() {
	log.Infof("Clearing all cloud data in 5 seconds")

	time.Sleep(time.Second * 5)

	c.ThingModel.ClearCloud()
	c.ChannelModel.ClearCloud()
	c.DeviceModel.ClearCloud()
	c.RoomModel.ClearCloud()
	c.SiteModel.ClearCloud()

	log.Infof("All cloud data cleared? Probably.")
}

func (c *HomeCloud) ExportRPCServices() {
	c.Conn.MustExportService(c.ThingModel, "$home/services/ThingModel", &model.ServiceAnnouncement{
		Schema: "/service/thing-model",
	})
	c.Conn.MustExportService(c.DeviceModel, "$home/services/DeviceModel", &model.ServiceAnnouncement{
		Schema: "/service/device-model",
	})
	c.Conn.MustExportService(c.RoomModel, "$home/services/RoomModel", &model.ServiceAnnouncement{
		Schema: "/service/room-model",
	})
	c.Conn.MustExportService(c.SiteModel, "$home/services/SiteModel", &model.ServiceAnnouncement{
		Schema: "/service/site-model",
	})
}

func (c *HomeCloud) StartSyncing(interval time.Duration) chan bool {

	syncComplete := make(chan bool)

	go func() {
		for {

			log.Infof("\n\n\n------ Timed model syncing started (every %s) ------ ", interval.String())

			roomResult := c.RoomModel.Sync()
			deviceResult := c.DeviceModel.Sync()
			channelResult := c.ChannelModel.Sync()
			thingResult := c.ThingModel.Sync()
			siteResult := c.SiteModel.Sync()

			success := true

			if roomResult != nil {
				log.Infof("Room sync error: %s", roomResult)
				success = false
			}
			if deviceResult != nil {
				log.Infof("Device sync error: %s", deviceResult)
				success = false
			}
			if channelResult != nil {
				log.Infof("Channel sync error: %s", channelResult)
				success = false
			}
			if thingResult != nil {
				log.Infof("Thing sync error: %s", thingResult)
				success = false
			}
			if siteResult != nil {
				log.Infof("Site sync error: %s", siteResult)
				success = false
			}

			log.Infof("------ Timed model syncing complete ------\n\n\n")

			select {
			case syncComplete <- success:
			default:
			}

			time.Sleep(interval)
		}
	}()

	return syncComplete
}

func (c *HomeCloud) AutoStartModules() {

	do := func(name string, task string) error {
		return c.Conn.SendNotification("$node/"+config.Serial()+"/module/"+task, name)
	}

	interval := config.MustDuration("homecloud.autoStart.duration")

	for _, name := range config.MustStringArray("homecloud.autoStart.modules") {
		log.Infof("-- (Re)starting '%s'", name)

		err := do(name, "stop")
		if err != nil {
			log.Fatalf("Failed to send %s stop message! %s", name, err)
		}

		time.Sleep(interval)

		err = do(name, "start")
		if err != nil {
			log.Fatalf("Failed to send %s start message! %s", name, err)
		}
	}

}

var locationRegexp = regexp.MustCompile("\\$device\\/([A-F0-9]*)\\/[^\\/]*\\/location")

type incomingLocationUpdate struct {
	Zone *string `json:"zone,omitempty"`
}

type outgoingLocationUpdate struct {
	ID         *string `json:"id"`
	HasChanged bool    `json:"hasChanged"`
}

/*
func Start(c *ninja.Connection) {

	startManagingDrivers()
	startManagingApps()
	startManagingDevices()
	startMonitoringLocations()
	startManagingTimeSeries()

	ensureNodeDeviceExists()
	ensureSiteExists()

	go func() {
		// Give it a chance to sync first...
		time.Sleep(time.Second * 20)
		//startDrivers()
		startApps()
	}()

	ledController := conn.GetServiceClient("$node/" + config.Serial() + "/led-controller")
	go func() {
		for {
			ledController.Call("enableControl", nil, nil, 0)
			time.Sleep(time.Second * 5)
		}
	}()

}

func ensureSiteExists() {
	site, err := siteModel.Fetch(config.MustString("siteId"))
	if err != nil && err != RecordNotFound {
		log.Fatalf("Failed to get site: %s", err)
	}

	if err == RecordNotFound {
		siteType := "home"
		site = &model.Site{
			ID:   config.MustString("siteId"),
			Type: &siteType,
		}
		err = siteModel.Create(site)
		if err != nil && err != RecordNotFound {
			log.Fatalf("Failed to create site: %s", err)
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

func startApp(node string, appID string, config *string) error {

	var rawConfig json.RawMessage
	if config != nil {
		rawConfig = []byte(*config)
	} else {
		rawConfig = []byte("{}")
	}

	client := conn.GetServiceClient(fmt.Sprintf("$node/%s/app/%s", node, appID))
	err := client.Call("start", &rawConfig, nil, 10*time.Second)

	if err != nil {
		jsonError, ok := err.(*json2.Error)
		if ok {
			if jsonError.Code == json2.E_INVALID_REQ {

				err := appModel.DeleteConfig(appID)
				if err != nil {
					log.Warningf("App %s could not parse its config. Also, we couldn't clear it! errors:%s and %s", appID, jsonError.Message, err)
				} else {
					log.Warningf("App %s could not parse its config, so we cleared it from redis. error:%s", appID, jsonError.Message)
				}

				return startApp(node, appID, nil)
			}
		}
	}

	return err
}

func startManagingDrivers() {

	conn.Subscribe("$node/:node/driver/:driver/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		node, driver := values["node"], values["driver"]

		log.Infof("Got driver announcement node:%s driver:%s", node, driver)

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

func startManagingApps() {

	conn.Subscribe("$node/:node/app/:app/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		node, app := values["node"], values["app"]

		log.Infof("Got app announcement node:%s app:%s", node, app)

		if announcement == nil {
			log.Warningf("Nil app announcement from node:%s app:%s", node, app)
			return true
		}

		module := &model.Module{}
		err := json.Unmarshal(*announcement, module)

		if announcement == nil {
			log.Warningf("Could not parse announcement from node:%s app:%s error:%s", node, app, err)
			return true
		}

		err = appModel.Create(module)
		if err != nil {
			log.Warningf("Failed to save app announcement for %s error:%s", app, err)
		}

		config, err := appModel.GetConfig(values["app"])

		if err != nil {
			log.Warningf("Failed to retrieve config for app %s error:%s", app, err)
		} else {
			err = startApp(node, app, config)
			if err != nil {
				log.Warningf("Failed to start app: %s error:%s", app, err)
			}
		}

		return true
	})

	conn.Subscribe("$node/:node/app/:app/event/config", func(config *json.RawMessage, values map[string]string) bool {
		log.Infof("Got app config node:%s app:%s config:%s", values["node"], values["app"], *config)

		if config != nil {
			err := appModel.SetConfig(values["app"], string(*config))

			if err != nil {
				log.Warningf("Failed to save config for app: %s error: %s", values["app"], err)
			}
		} else {
			log.Infof("Nil config recevied from node:%s app:%s", values["node"], values["app"])
		}

		return true
	})

}

func startManagingDevices() {

	conn.Subscribe("$device/:id/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		id := values["id"]

		log.Infof("Got device announcement device:%s", id)

		if announcement == nil {
			log.Warningf("Nil device announcement from device:%s", id)
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

		log.Infof("Got channel announcement device:%s channel:%s", deviceID, channelID)

		if announcement == nil {
			log.Warningf("Nil channel announcement from device:%s channel:%s", deviceID, channelID)
			return true
		}

		channel := &model.Channel{}
		err := json.Unmarshal(*announcement, channel)

		if err != nil {
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

	_, err := conn.GetMqttClient().Subscribe("$device/+/+/location", func(topic string, payload []byte) {

		deviceID := locationRegexp.FindAllStringSubmatch(topic, -1)[0][1]

		update := &incomingLocationUpdate{}
		err := json.Unmarshal(payload, update)
		if err != nil {
			log.Errorf("Failed to parse location update %s to %s : %s", payload, topic, err)
			return
		}

		thing, err := thingModel.FetchByDeviceId(deviceID)
		if err != nil && err != RecordNotFound {
			log.Warningf("Failed to fetch thing by device id %s", deviceID)
		}

		if update.Zone == nil {
			log.Debugf("< Incoming location update: device %s not in a zone", deviceID)
		} else {
			log.Debugf("< Incoming location update: device %s is in zone %s", deviceID, *update.Zone)
		}

		hasChangedZone := true

		if err == RecordNotFound {
			log.Debugf("Device %s is not attached to a thing. Ignoring.", deviceID)
		} else {

			if (thing.Location != nil && update.Zone != nil && *thing.Location == *update.Zone) || (thing.Location == nil && update.Zone == nil) {
				// It's already there
				log.Debugf("Thing %s (%s) (Device %s) was already in that zone.", thing.ID, thing.Name, deviceID)
				hasChangedZone = true
			} else {

				log.Debugf("Thing %s (%s) (Device %s) moved from %s to %s", thing.ID, thing.Name, deviceID, thing.Location, update.Zone)

				err = thingModel.SetLocation(thing.ID, update.Zone)
				if err != nil {
					log.FatalError(err, fmt.Sprintf("Failed to update location property of thing %s", thing.ID))
				}

				if update.Zone != nil {
					_, err := roomModel.Fetch(*update.Zone)
					if err != nil && err != RecordNotFound {
						log.FatalError(err, fmt.Sprintf("Failed to fetch room %s", *update.Zone))
					}

					if err != RecordNotFound {
						// XXX: TODO: Remove me once the cloud room model is sync'd and locatino service uses it
						log.Infof("Unknown room %s. Advising remote location service to forget it.", *update.Zone)

						conn.GetMqttClient().Publish("$location/delete", payload)
					}
				}
			}

			topic := fmt.Sprintf("$device/%s/channel/%s/%s/event/state", deviceID, "location", "location")

			payload, _ := json.Marshal(&outgoingLocationUpdate{
				ID:         update.Zone,
				HasChanged: hasChangedZone,
			})

			conn.GetMqttClient().Publish(topic, payload)

		}

	})

	if err != nil {
		log.Fatalf("Failed to subscribe to device locations: %s", err)
	}

}
*/
