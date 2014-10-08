package homecloud

import (
	"fmt"
	"reflect"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type ThingModel struct {
	baseModel
}

func NewThingModel(pool *redis.Pool, conn *ninja.Connection) *ThingModel {
	return &ThingModel{baseModel{pool, "thing", reflect.TypeOf(model.Thing{}), conn, logger.GetLogger("ThingModel")}}
}

func (m *ThingModel) MustSync() {
	if err := m.sync(); err != nil {
		m.log.Fatalf("Failed to sync %ss error:%s", m.idType, err)
	}
}

func (m *ThingModel) Create(thing *model.Thing) error {
	if thing.ID == "" {
		thing.ID = uuid.NewUUID().String()
	}

	_, err := m.save(thing.ID, thing)
	return err
}

func (m *ThingModel) Delete(id string) error {

	// TODO: announce deletion via MQTT
	// self.bus.publish(Ninja.topics.thing.goodbye.thing(thing.id), {id: thing.id});

	return m.delete(id)
}

func (m *ThingModel) FetchByDeviceId(deviceId string) (*model.Thing, error) {
	device, err := deviceModel.Fetch(deviceId)
	if err != nil {
		return nil, err
	}

	if device.Thing == nil {
		return nil, nil
	}

	return m.Fetch(*device.Thing)
}

func (m *ThingModel) SetLocation(thingID string, roomID *string) error {

	conn := m.pool.Get()
	defer conn.Close()

	var err error

	if roomID == nil {
		_, err = conn.Do("HDEL", "thing:"+thingID, "location")
	} else {
		_, err = conn.Do("HSET", "thing:"+thingID, "location", *roomID)
	}

	return err
}

func (m *ThingModel) Fetch(id string) (*model.Thing, error) {
	thing := &model.Thing{}

	if err := m.fetch(id, thing); err != nil {
		return nil, fmt.Errorf("Failed to fetch thing (id:%s): %s", id, err)
	}

	if thing.DeviceID != nil {
		device, err := deviceModel.Fetch(*thing.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch nested device (id:%s) : %s", *thing.DeviceID, err)
		}
		thing.Device = device
	}

	return thing, nil
}

func (m *ThingModel) FetchByType(thingType string) (*[]*model.Thing, error) {
	allThings, err := m.FetchAll()

	if err != nil {
		return nil, err
	}

	filtered := []*model.Thing{}

	for _, thing := range *allThings {
		if thing.Type == thingType {
			filtered = append(filtered, thing)
		}
	}

	return &filtered, nil
}

func (m *ThingModel) FetchAll() (*[]*model.Thing, error) {

	ids, err := m.fetchIds()

	if err != nil {
		return nil, err
	}

	things := make([]*model.Thing, len(ids))

	for i, id := range ids {
		things[i], err = m.Fetch(id)
		if err != nil {
			return nil, err
		}
	}

	return &things, nil
}

// Update a thing, this is currently very optimisic and only changes name and type fields.
func (m *ThingModel) Update(id string, thing *model.Thing) error {

	oldThing := &model.Thing{}

	if err := m.fetch(id, thing); err != nil {
		return fmt.Errorf("Failed to fetch thing (id:%s): %s", id, err)
	}

	oldThing.Name = thing.Name
	oldThing.Type = thing.Type

	if _, err := m.save(id, oldThing); err != nil {
		return fmt.Errorf("Failed to update thing (id:%s): %s", id, err)
	}

	return nil
}
