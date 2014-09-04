package homecloud

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	conn redis.Conn
}

func NewDeviceModel(conn redis.Conn) *DeviceModel {
	return &DeviceModel{conn}
}

func (m *DeviceModel) Fetch(id string) (*model.Device, error) {

	item, err := redis.Values(m.conn.Do("HGETALL", "device:"+id))

	if err != nil {
		return nil, err
	}

	device := &model.Device{}

	if err := redis.ScanStruct(item, device); err != nil {
		return nil, err
	}

	return device, nil
}
