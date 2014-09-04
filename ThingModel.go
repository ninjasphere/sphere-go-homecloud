package homecloud

import (
	"fmt"

	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type ThingModel struct {
	conn redis.Conn
}

func NewThingModel(conn redis.Conn) *ThingModel {
	return &ThingModel{conn}
}

func (m *ThingModel) FetchByDeviceId(deviceId string) (*model.Thing, error) {
	device, err := deviceModel.Fetch(deviceId)
	if err != nil {
		return nil, err
	}

	if device.Thing == nil {
		return nil, fmt.Errorf("Device %s is not attached to a thing", deviceId)
	}

	return m.Fetch(*device.Thing)
}

func (m *ThingModel) Fetch(id string) (*model.Thing, error) {

	item, err := redis.Values(m.conn.Do("HGETALL", "thing:"+id))

	if err != nil {
		return nil, err
	}

	thing := &model.Thing{}

	if err := redis.ScanStruct(item, thing); err != nil {
		return nil, err
	}

	if thing.DeviceId != nil {
		device, err := deviceModel.Fetch(*thing.DeviceId)
		if err != nil {
			return nil, err
		}
		thing.Device = device
	}

	return thing, nil
}