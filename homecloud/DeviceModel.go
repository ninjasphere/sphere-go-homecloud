package homecloud

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DeviceModel struct {
	baseModel
}

func NewDeviceModel(conn redis.Conn) *DeviceModel {
	return &DeviceModel{baseModel{conn, "device"}}
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
