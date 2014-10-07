package homecloud

import (
	"reflect"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DriverModel struct {
	baseModel
}

func NewDriverModel(pool *redis.Pool, conn *ninja.Connection) *DriverModel {
	return &DriverModel{baseModel{pool, "driver", reflect.TypeOf(model.Module{}), conn}}
}

func (m *DriverModel) Fetch(id string) (*model.Module, error) {

	module := &model.Module{}

	if err := m.fetch(id, module); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *DriverModel) Create(module *model.Module) error {
	_, err := m.save(module.ID, module)
	return err
}

func (m *DriverModel) GetConfig(driverID string) (*string, error) {

	conn := m.pool.Get()
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("HEXISTS", "driver:"+driverID, "config"))

	if exists {
		item, err := conn.Do("HGET", "driver:"+driverID, "config")
		config, err := redis.String(item, err)
		return &config, err
	}

	return nil, err
}

func (m *DriverModel) SetConfig(driverID string, config string) error {

	conn := m.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HSET", "driver:"+driverID, "config", config)
	return err
}

func (m *DriverModel) DeleteConfig(driverID string) error {

	conn := m.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HDEL", "driver:"+driverID, "config")
	return err
}
