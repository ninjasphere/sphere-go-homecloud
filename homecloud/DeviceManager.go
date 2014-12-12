package homecloud

import (
	"encoding/json"
	"fmt"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

type DeviceManager struct {
	Conn         *ninja.Connection    `inject:""`
	DeviceModel  *models.DeviceModel  `inject:""`
	ChannelModel *models.ChannelModel `inject:""`
	log          *logger.Logger
}

func (m *DeviceManager) PostConstruct() error {
	m.log = logger.GetLogger("DeviceManager")
	err := m.Start()
	if err != nil {
		return err
	}
	m.exportNodeDevice()
	return nil
}

func (m *DeviceManager) exportNodeDevice() error {

	device := &NodeDevice{ninja.LoadModuleInfo("./package.json")}

	err := m.Conn.ExportDevice(device)
	if err != nil {
		return fmt.Errorf("Failed to export node device: %s", err)
	}
	return nil
}

func (m *DeviceManager) Start() error {

	// Listen for device announcements, and save them to redis
	m.Conn.Subscribe("$device/:id/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

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

		err = m.DeviceModel.Create(device)
		if err != nil {
			log.Warningf("Failed to save device announcement for device:%s error:%s", id, err)
		}

		return true
	})

	// Listen for channel announcements, and save them to redis
	return m.Conn.Subscribe("$device/:device/channel/:channel/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

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

		err = m.ChannelModel.Create(deviceID, channel)
		if err != nil {
			log.Warningf("Failed to save channel announcement for device:%s channel:%s error:%s", deviceID, channelID, err)
		}

		return true
	})
}
