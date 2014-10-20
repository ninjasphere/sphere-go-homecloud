package homecloud

import (
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/model"
)

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
