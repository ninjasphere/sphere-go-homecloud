package homecloud

import (
	"reflect"
	"sync"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

// TODO: Sync app config

type AppModel struct {
	baseModel
}

func NewAppModel(pool *redis.Pool, conn *ninja.Connection) *AppModel {
	return &AppModel{
		baseModel{
			syncing: &sync.WaitGroup{},
			pool:    pool,
			idType:  "app",
			objType: reflect.TypeOf(model.Module{}),
			conn:    conn,
			log:     logger.GetLogger("AppModel"),
			sendEvent: func(event string, payload interface{}) error {
				// Not currently exposed as a service
				return nil
			},
		},
	}
}

func (m *AppModel) Fetch(id string) (*model.Module, error) {
	m.syncing.Wait()
	//defer m.sync()

	module := &model.Module{}

	if err := m.fetch(id, module, false); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *AppModel) Create(module *model.Module) error {
	m.syncing.Wait()
	//defer m.sync()

	_, err := m.save(module.ID, module)
	return err
}

func (m *AppModel) GetConfig(appID string) (*string, error) {
	m.syncing.Wait()

	conn := m.pool.Get()
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("HEXISTS", "app:"+appID, "config"))

	if exists {
		item, err := conn.Do("HGET", "app:"+appID, "config")
		config, err := redis.String(item, err)
		return &config, err
	}

	return nil, err
}

func (m *AppModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	err := m.delete(id)
	if err != nil {
		return err
	}

	return m.DeleteConfig(id)
}

func (m *AppModel) SetConfig(appID string, config string) error {
	m.syncing.Wait()
	//defer m.sync()

	conn := m.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HSET", "app:"+appID, "config", config)
	return err
}

func (m *AppModel) DeleteConfig(appID string) error {
	m.syncing.Wait()
	//defer m.sync()

	conn := m.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HDEL", "app:"+appID, "config")
	return err
}
