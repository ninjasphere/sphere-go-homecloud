package models

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

// TODO: Sync driver config

type DriverModel struct {
	baseModel
}

func NewDriverModel() *DriverModel {
	model := &DriverModel{
		baseModel: newBaseModel("driver", model.Module{}),
	}
	model.sendEvent = func(event string, payload interface{}) error {
		// Not currently exposed as a service
		return nil
	}
	return model
}

func (m *DriverModel) Fetch(id string) (*model.Module, error) {
	m.syncing.Wait()
	//defer m.sync()

	module := &model.Module{}

	if err := m.fetch(id, module, false); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *DriverModel) Create(module *model.Module) error {
	m.syncing.Wait()
	//defer m.sync()

	_, err := m.save(module.ID, module)
	return err
}

func (m *DriverModel) GetConfig(driverID string) (*string, error) {
	m.syncing.Wait()

	conn := m.Pool.Get()
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("HEXISTS", "driver:"+driverID, "config"))

	if exists {
		item, err := conn.Do("HGET", "driver:"+driverID, "config")
		config, err := redis.String(item, err)
		return &config, err
	}

	return nil, err
}

func (m *DriverModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	err := m.delete(id)
	if err != nil {
		return err
	}

	return m.DeleteConfig(id)
}

func (m *DriverModel) SetConfig(driverID string, config string) error {
	m.syncing.Wait()
	//defer m.sync()

	conn := m.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("HSET", "driver:"+driverID, "config", config)
	return err
}

func (m *DriverModel) DeleteConfig(driverID string) error {
	m.syncing.Wait()
	//defer m.sync()

	conn := m.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("HDEL", "driver:"+driverID, "config")
	return err
}
