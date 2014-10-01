package homecloud

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DriverModel struct {
	baseModel
}

func NewDriverModel(conn redis.Conn) *DriverModel {
	return &DriverModel{baseModel{conn, "driver"}}
}

func (m *DriverModel) Fetch(id string) (*model.Module, error) {

	module := &model.Module{}

	if err := m.fetch(id, module); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *DriverModel) Create(module *model.Module) error {
	return m.create(module.ID, module)
}

func (m *DriverModel) GetConfig(driverID string) (*string, error) {
	exists, err := redis.Bool(m.conn.Do("HEXISTS", "driver:"+driverID, "config"))

	if exists {
		item, err := m.conn.Do("HGET", "driver:"+driverID, "config")
		config, err := redis.String(item, err)
		return &config, err
	}

	return nil, err
}

func (m *DriverModel) SetConfig(driverID string, config string) error {
	_, err := m.conn.Do("HSET", "driver:"+driverID, "config", config)
	return err
}

func (m *DriverModel) DeleteConfig(driverID string) error {
	_, err := m.conn.Do("HDEL", "driver:"+driverID, "config")
	return err
}
