package homecloud

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
	channelModel baseModel
}

func NewDeviceModel(conn redis.Conn) *DeviceModel {
	return &DeviceModel{
		baseModel{conn, "device"},
		baseModel{conn, "channel"},
	}
}

func (m *DeviceModel) Fetch(id string) (*model.Device, error) {

	device := &model.Device{}

	if err := m.fetch(id, device); err != nil {
		return nil, err
	}

	return device, nil
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
