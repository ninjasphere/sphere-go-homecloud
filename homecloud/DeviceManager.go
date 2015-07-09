package homecloud

import (
	"encoding/json"
	"fmt"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

type DeviceManager struct {
	Conn         *ninja.Connection    `inject:""`
	DeviceModel  *models.DeviceModel  `inject:""`
	ChannelModel *models.ChannelModel `inject:""`
	ThingModel   *models.ThingModel   `inject:""`
	Pool         *redis.Pool          `inject:""`
	log          *logger.Logger
}

func (m *DeviceManager) PostConstruct() error {
	m.log = logger.GetLogger("DeviceManager")
	err := m.Start()
	if err != nil {
		return err
	}
	return nil
}

func (m *DeviceManager) Start() error {

	// Listen for device announcements, and save them to redis
	_, err := m.Conn.Subscribe("$device/:id/event/announce", func(announcement *json.RawMessage, values map[string]string) bool {

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

		conn := m.Pool.Get()
		defer conn.Close()

		err = m.DeviceModel.Create(device, conn)
		if err != nil {
			log.Warningf("Failed to save device announcement for device:%s error:%s", id, err)
		}

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

		conn := m.Pool.Get()
		defer conn.Close()
		err = m.ChannelModel.Create(deviceID, channel, conn)
		if err != nil {
			log.Warningf("Failed to save channel announcement for device:%s channel:%s error:%s", deviceID, channelID, err)
		}

		return true
	})

	if err != nil {
		return err
	}

	/*// Map device events to thing events
	_, err = m.Conn.SubscribeRaw("$device/:device/channel/:channel/event/:event", func(payload *json.RawMessage, values map[string]string) bool {

		if values["event"] == "announce" {
			// We don't care about announcements
			return true
		}

		conn := m.Pool.Get()
		defer conn.Close()
		thing, err := m.ThingModel.GetThingIDForDevice(values["device"], conn)
		if err != nil {
			log.Errorf("Got an event, but failed to fetch the thing id for device: %s error: %s", values["device"], err)
			return true
		}

		m.Conn.GetMqttClient().Publish(fmt.Sprintf("$thing/%s/channel/%s/event/%s", *thing, values["channel"], values["event"]), *payload)

		return true
	})*/

	if err != nil {
		return err
	}

	// Map thing actuations to device actuations
	_, err = m.Conn.SubscribeRaw("$thing/:thing/channel/:channel", func(payload *json.RawMessage, values map[string]string) bool {

		conn := m.Pool.Get()
		defer conn.Close()
		device, err := m.ThingModel.GetDeviceIDForThing(values["thing"], conn)
		if err != nil {
			log.Errorf("Got a thing actuation, but failed to fetch the device for thing: %s error: %s", values["thing"], err)
			return true
		}

		m.Conn.GetMqttClient().Publish(fmt.Sprintf("$device/%s/channel/%s", *device, values["channel"]), *payload)

		return true
	})

	if err != nil {
		return err
	}

	// Map device actuation replies to thing actuation replies
	_, err = m.Conn.SubscribeRaw("$device/:device/channel/:channel/reply", func(payload *json.RawMessage, values map[string]string) bool {

		conn := m.Pool.Get()
		defer conn.Close()
		thing, err := m.ThingModel.GetThingIDForDevice(values["device"], conn)
		if err != nil {
			log.Errorf("Got a device actuation reply, but failed to fetch the thing for device: %s error: %s", values["device"], err)
			return true
		}

		m.Conn.GetMqttClient().Publish(fmt.Sprintf("$thing/%s/channel/%s/reply", *thing, values["channel"]), *payload)

		return true
	})

	return err
}
