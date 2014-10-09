package homecloud

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
}

/*
pool    *redis.Pool
idType  string
objType reflect.Type
conn    *ninja.Connection
log     *logger.Logger
*/

func NewDeviceModel(pool *redis.Pool, conn *ninja.Connection) *DeviceModel {
	return &DeviceModel{
		baseModel{
			syncing: &sync.WaitGroup{},
			pool:    pool,
			idType:  "device",
			objType: reflect.TypeOf(model.Device{}),
			conn:    conn,
			log:     logger.GetLogger("DeviceModel"),
		},
	}
}

func (m *DeviceModel) Fetch(deviceID string) (*model.Device, error) {
	m.syncing.Wait()

	device := &model.Device{}

	if err := m.fetch(deviceID, device, false); err != nil {
		return nil, err
	}

	channels, err := channelModel.FetchAll(deviceID)

	if err != nil {
		return nil, fmt.Errorf("Failed to get channels for device id:%s error:%s", deviceID, err)
	}

	device.Channels = channels

	thingID, err := thingModel.getThingIDForDevice(deviceID)

	if err != nil && err != RecordNotFound {
		return nil, err
	}

	device.ThingID = thingID

	return device, nil
}

func (m *DeviceModel) FetchAll() (*[]*model.Device, error) {
	m.syncing.Wait()

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
	m.syncing.Wait()
	//defer m.sync()

	log.Debugf("Saving device %s", device.ID)

	existing, err := m.Fetch(device.ID)

	if err != nil && err != RecordNotFound {
		return err
	}

	updated, err := m.save(device.ID, device)

	log.Debugf("Device was updated? %t", updated)

	if err != nil || existing != nil {
		return err
	}

	// It was a new device, we might need to make a thing for it
	thing, err := thingModel.FetchByDeviceId(device.ID)

	if err != nil && err != RecordNotFound {
		// An actual error
		return err
	}

	if thing != nil {
		// A thing already exists
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

	return thingModel.Create(thing)
}

func (m *DeviceModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	err := m.delete(id)
	if err != nil {
		err = thingModel.deleteRelationshipWithDevice(id)
	}
	return err
}
