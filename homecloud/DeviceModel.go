package homecloud

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
}

func NewDeviceModel(conn redis.Conn) *DeviceModel {
	return &DeviceModel{
		baseModel{conn, "device"},
	}
}

func (m *DeviceModel) Fetch(deviceID string) (*model.Device, error) {

	device := &model.Device{}

	if err := m.fetch(deviceID, device); err != nil {
		return nil, err
	}

	channelIds, err := redis.Strings(m.conn.Do("SMEMBERS", "device:"+deviceID+":channels"))

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
	return m.create(device.ID, device)
}

func (m *DeviceModel) AddChannel(channel *model.Channel) error {

	args := redis.Args{}
	args = args.Add("device:" + channel.Device.ID + ":channel:" + channel.ID)
	args = args.AddFlat(channel)

	if _, err := m.conn.Do("HMSET", args...); err != nil {
		return err
	}

	if _, err := m.conn.Do("SADD", "device:"+channel.Device.ID+":channels", channel.ID); err != nil {
		return err
	}

	return nil
}

func (m *baseModel) getChannel(deviceID, channelID string) (*model.Channel, error) {

	item, err := redis.Values(m.conn.Do("HGETALL", "device:"+deviceID+":channel:"+channelID))

	if err != nil {
		return nil, err
	}

	channel := &model.Channel{}

	if err := redis.ScanStruct(item, channel); err != nil {
		return nil, err
	}

	return channel, nil
}
