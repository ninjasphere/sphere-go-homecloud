package homecloud

import (
	"reflect"
	"sort"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
	channelModel baseModel
}

func NewDeviceModel(pool *redis.Pool, conn *ninja.Connection) *DeviceModel {
	return &DeviceModel{
		baseModel{pool, "device", reflect.TypeOf(model.Device{}), conn},
		baseModel{pool, "channel", reflect.TypeOf(model.Channel{}), conn},
	}
}

func (m *DeviceModel) Fetch(deviceID string) (*model.Device, error) {

	device := &model.Device{}

	if err := m.fetch(deviceID, device); err != nil {
		return nil, err
	}

	conn := m.pool.Get()
	defer conn.Close()

	channelIds, err := redis.Strings(conn.Do("SMEMBERS", "device:"+deviceID+":channels"))

	if err != nil {
		return nil, err
	}

	channels := make([]*model.Channel, len(channelIds))

	for i, channelID := range channelIds {
		channels[i], err = m.getChannel(deviceID, channelID)
		if err != nil {
			return nil, err
		}
	}

	device.Channels = &channels

	return device, nil
}

func (m *DeviceModel) FetchAll() (*[]*model.Device, error) {

	ids, err := m.fetchAllIds()

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

func (m *DeviceModel) AddChannel(channel *model.Channel) error {

	// First.. make sure the string arrays are sorted :/
	sort.Strings(*channel.SupportedMethods)
	sort.Strings(*channel.SupportedEvents)

	updated, err := m.channelModel.saveWithRoot("device:"+channel.Device.ID+":channel", channel.ID, channel)
	if err != nil {
		return err
	}
	log.Debugf("Channel was updated? %t", updated)
	if updated {
		return m.markUpdated("device", channel.Device.ID, time.Now())
	}
	return nil
}

func (m *baseModel) getChannel(deviceID, channelID string) (*model.Channel, error) {

	conn := m.pool.Get()
	defer conn.Close()

	item, err := redis.Values(conn.Do("HGETALL", "device:"+deviceID+":channel:"+channelID))

	if err != nil {
		return nil, err
	}

	channel := &model.Channel{}

	if err := redis.ScanStruct(item, channel); err != nil {
		return nil, err
	}

	return channel, nil
}
