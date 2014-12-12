package homecloud

import "regexp"

/*
// TODO: This won't work with multiple spheres connected to this homecloud.
func ensureNodeDeviceExists() {
	exists, err := thingModel.Exists(config.Serial())
	if err != nil {
		log.Errorf("Failed checking if node device exists: %s", err)
	}
	if exists {
		return
	}

	device := &NodeDevice{ninja.LoadModuleInfo("./package.json")}

	err = conn.ExportDevice(device)
	if err != nil {
		log.Errorf("Failed to export node device: %s", err)
	}
}

type NodeDevice struct {
	info *model.Module
}

func (d *NodeDevice) GetDeviceInfo() *model.Device {
	name := "Spheramid " + config.Serial()
	return &model.Device{
		NaturalID:     config.Serial(),
		NaturalIDType: "node",
		Name:          &name,
		Signatures: &map[string]string{
			"ninja:manufacturer": "Ninja Blocks Inc.",
			"ninja:productName":  "Spheramid",
			"ninja:thingType":    "node",
		},
	}
}

func (d *NodeDevice) GetModuleInfo() *model.Module {
	return d.info
}

func (d *NodeDevice) GetDriver() ninja.Driver {
	return d
}

func (d *NodeDevice) SetEventHandler(handler func(event string, payload interface{}) error) {
}

/*
func NewFakeDriver() (*FakeDriver, error) {

	driver := &FakeDriver{}

	err := driver.Init(info)
	if err != nil {
		log.Fatalf("Failed to initialize fake driver: %s", err)
	}

	err = driver.Export(driver)
	if err != nil {
		log.Fatalf("Failed to export fake driver: %s", err)
	}

	userAgent := driver.Conn.GetServiceClient("$device/:deviceId/channel/user-agent")
	userAgent.OnEvent("pairing-requested", driver.OnPairingRequest)

	return driver, nil
}*/

var locationRegexp = regexp.MustCompile("\\$device\\/([A-F0-9]*)\\/[^\\/]*\\/location")

type incomingLocationUpdate struct {
	Zone *string `json:"zone,omitempty"`
}

type outgoingLocationUpdate struct {
	ID         *string `json:"id"`
	HasChanged bool    `json:"hasChanged"`
}

/*

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
