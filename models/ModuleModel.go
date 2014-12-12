package models

import (
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

// TODO: Sync module config

type ModuleModel struct {
	baseModel
}

func NewModuleModel() *ModuleModel {
	model := &ModuleModel{
		baseModel: newBaseModel("module", model.Module{}),
	}
	model.sendEvent = func(event string, payload interface{}) error {
		// Not currently exposed as a service
		return nil
	}
	return model
}

func (m *ModuleModel) Fetch(id string) (*model.Module, error) {
	m.syncing.Wait()
	//defer m.sync()

	module := &model.Module{}

	if err := m.fetch(id, module, false); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *ModuleModel) Create(module *model.Module) error {
	m.syncing.Wait()
	//defer m.sync()

	_, err := m.save(module.ID, module)
	return err
}

func (m *ModuleModel) GetConfig(moduleID string) (*string, error) {
	m.syncing.Wait()

	conn := m.Pool.Get()
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("HEXISTS", "module:"+moduleID, "config"))

	if exists {
		item, err := conn.Do("HGET", "module:"+moduleID, "config")
		config, err := redis.String(item, err)
		return &config, err
	}

	return nil, err
}

func (m *ModuleModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	err := m.delete(id)
	if err != nil {
		return err
	}

	return m.DeleteConfig(id)
}

func (m *ModuleModel) SetConfig(moduleID string, config string) error {
	m.syncing.Wait()
	//defer m.sync()

	conn := m.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("HSET", "module:"+moduleID, "config", config)
	return err
}

func (m *ModuleModel) DeleteConfig(moduleID string) error {
	m.syncing.Wait()
	//defer m.sync()

	conn := m.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("HDEL", "module:"+moduleID, "config")
	return err
}
