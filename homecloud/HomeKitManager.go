package homecloud

import (
	"encoding/json"

	"github.com/brutella/hc/hap"
	hcmodel "github.com/brutella/hc/model"
	"github.com/brutella/hc/model/accessory"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
)

type HomeKitManager struct {
	Conn *ninja.Connection `inject:""`
	log  *logger.Logger

	transport   hap.Transport
	accessories map[string]hcmodel.Accessory
	channels    map[string]*model.Channel
}

func (m *HomeKitManager) PostConstruct() error {
	m.log = logger.GetLogger("HomeKitManager")
	err := m.Start()
	if err != nil {
		return err
	}

	m.transport, err = hap.NewIPTransport("11223344", accessory.New(hcmodel.Info{
		Name:         "Ninja Sphere",
		SerialNumber: config.Serial(),
		Manufacturer: "Ninja Blocks",
		Model:        "Sphere",
		Firmware:     "1.0.awesome", // TODO: Use real values for this stuff
		Software:     "1.0.so.amazing",
		Hardware:     "1.0.much.hardware",
	}))

	return err
}

func (m *HomeKitManager) Start() error {

	// Listen for device announcements, and expose them to HomeKit
	_, err := m.Conn.Subscribe("$device/:id/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

		id := values["id"]

		m.log.Infof("Got device announcement device:%s", id)

		if announcement == nil {
			m.log.Warningf("Nil device announcement from device:%s", id)
			return true
		}

		device := &model.Device{}
		err := json.Unmarshal(*announcement, device)

		if announcement == nil {
			m.log.Warningf("Could not parse announcement from device:%s error:%s", id, err)
			return true
		}

		if _, ok := m.accessories[device.ID]; ok {
			// We've already seen this
			return true
		}

		info := hcmodel.Info{
			SerialNumber: device.ID,
		}

		if device.Name != nil {
			info.Name = *device.Name
		}

		if manufacturer, ok := (*device.Signatures)["ninja:manufacturer"]; ok {
			info.Manufacturer = manufacturer
		}

		if productName, ok := (*device.Signatures)["ninja:productName"]; ok {
			info.Model = productName
		}

		m.accessories[device.ID] = accessory.New(info)

		m.log.Infof("Created homekit accessory for device: %s (%s:%s) - %s", device.ID, device.NaturalIDType, device.NaturalID, device.Name)

		return true
	})

	if err != nil {
		return err
	}

	// Listen for channel announcements, and save them to redis
	_, err = m.Conn.Subscribe("$device/:device/channel/:channel/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

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

		return true
	})

	if err != nil {
		return err
	}

	// Map device events to thing events
	_, err = m.Conn.SubscribeRaw("$device/:device/channel/:channel/event/:event", func(payload *json.RawMessage, values map[string]string) bool {

		if values["event"] == "announce" {
			// We don't care about announcements
			return true
		}

		return true
	})

	return err
}
