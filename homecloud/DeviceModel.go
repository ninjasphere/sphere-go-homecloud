package homecloud

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
}

func NewDeviceModel(pool *redis.Pool) *DeviceModel {
	return &DeviceModel{
		baseModel{pool, "device"},
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
	log.Infof("Saving device")

	existing, err := m.Fetch(device.ID)

	if err != nil && err != RecordNotFound {
		return err
	}

	err = m.create(device.ID, device)

	if err != nil || existing != nil {
		return err
	}

	// It was a new device, we might need to make a thing for it

	thing, err := thingModel.FetchByDeviceId(device.ID)

	if err != nil && err != RecordNotFound {
		return nil
	}

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

	return m.create(device.ID, device)
}

func (m *DeviceModel) AddChannel(channel *model.Channel) error {

	conn := m.pool.Get()
	defer conn.Close()

	args := redis.Args{}
	args = args.Add("device:" + channel.Device.ID + ":channel:" + channel.ID)
	args = args.AddFlat(channel)

	if _, err := conn.Do("HMSET", args...); err != nil {
		return err
	}

	if _, err := conn.Do("SADD", "device:"+channel.Device.ID+":channels", channel.ID); err != nil {
		return err
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
