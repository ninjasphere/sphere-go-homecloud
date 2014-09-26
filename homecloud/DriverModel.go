package homecloud

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type DriverModel struct {
	conn redis.Conn
}

func NewDriverModel(conn redis.Conn) *DriverModel {
	return &DriverModel{conn}
}

func (m *DriverModel) Fetch(driverID string) (*model.Module, error) {

	item, err := redis.Values(m.conn.Do("HGETALL", "driver:"+driverID))

	if err != nil {
		return nil, err
	}

	if len(item) == 0 {
		return nil, nil
	}

	module := &model.Module{}

	if err := redis.ScanStruct(item, module); err != nil {
		return nil, err
	}

	return module, nil
}

func (m *DriverModel) Save(module *model.Module) error {

	args := redis.Args{}
	args = args.Add("driver:" + module.ID)
	args = args.AddFlat(module)

	spew.Dump(args)

	_, err := m.conn.Do("HMSET", args...)

	return err
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
