package homecloud

import (
	"fmt"
	"reflect"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
}

func NewDeviceModel(pool *redis.Pool, conn *ninja.Connection) *DeviceModel {
	return &DeviceModel{
		baseModel{pool, "device", reflect.TypeOf(model.Device{}), conn, logger.GetLogger("DeviceModel")},
	}
}

func (m *DeviceModel) MustSync() {
	if err := m.sync(); err != nil {
		m.log.Fatalf("Failed to sync devices error:%s", err)
	}
}

func (m *DeviceModel) Fetch(deviceID string) (*model.Device, error) {

	device := &model.Device{}

	if err := m.fetch(deviceID, device); err != nil {
		return nil, err
	}

	channels, err := channelModel.FetchAll(deviceID)

	if err != nil {
		return nil, fmt.Errorf("Failed to get channels for device id:%s error:%s", deviceID, err)
	}

	device.Channels = channels

	return device, nil
}

func (m *DeviceModel) FetchAll() (*[]*model.Device, error) {

	ids, err := m.fetchIds()

	if err != nil {
		return nil, err
	}

	devices := make([]*model.Device, len(ids))

	for i, id := range ids {
		devices[i], err = m.Fetch(id)
		if err != nil {
			return nil, err
		}
	}

	return &devices, nil
}

func (m *DeviceModel) Create(device *model.Device) error {
	log.Debugf("Saving device")

	existing, err := m.Fetch(device.ID)

	if err != nil && err != RecordNotFound {
		return err
	}

	if existing != nil {
		// This relationship can not be changed here.
		device.Thing = existing.Thing
	}

	updated, err := m.save(device.ID, device)

	log.Debugf("Device was updated? %t", updated)

	if err != nil || existing != nil {
		return err
	}

	// It was a new device, we might need to make a thing for it

	thing, err := thingModel.FetchByDeviceId(device.ID)

	if err != nil && err != RecordNotFound {
		return nil
	}

	log.Debugf("New device has no thing. Creating one")

	thing = &model.Thing{
		DeviceID: &device.ID,
		Name:     "New Thing",
		Type:     "unknown",
	}

	if device.Signatures != nil {
		thingType, hasThingType := (*device.Signatures)["ninja:thingType"]

		if hasThingType {
			thing.Type = thingType
			if device.Name == nil {
				thing.Name = "New " + thingType
			}
		}
	}

	if device.Name != nil {
		thing.Name = *device.Name
	}

	err = thingModel.Create(thing)

	if err != nil && err != RecordNotFound {
		return nil
	}

	device.Thing = &thing.ID

	_, err = m.save(device.ID, device)
	return err
}

func (m *DeviceModel) Delete(id string) error {
	return m.delete(id)
}
