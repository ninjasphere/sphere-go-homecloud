package homecloud

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type ThingModel struct {
	baseModel
}

func toThing(obj interface{}) *model.Thing {
	var thing, ok = obj.(*model.Thing)
	if !ok {
		spew.Dump("BAD THING", obj)
		panic("Non-'Thing' passed to a ThingModel handler")
	}
	return thing
}

func NewThingModel(pool *redis.Pool, conn *ninja.Connection) *ThingModel {

	thingModel := &ThingModel{}
	thingModel.baseModel = baseModel{
		syncing: &sync.WaitGroup{},
		pool:    pool,
		idType:  "thing",
		objType: reflect.TypeOf(model.Thing{}),
		conn:    conn,
		log:     logger.GetLogger("ThingModel"),
		afterSave: func(obj interface{}) error {
			return thingModel.afterSave(toThing(obj))
		},
		beforeDelete: func(id string) error {
			return thingModel.beforeDelete(id)
		},
		onFetch: func(obj interface{}) error {
			return thingModel.onFetch(toThing(obj))
		},
	}
	return thingModel
}

func (m *ThingModel) Create(thing *model.Thing) error {
	m.syncing.Wait()
	//defer m.sync()

	if thing.ID == "" {
		thing.ID = uuid.NewUUID().String()
	}

	_, err := m.save(thing.ID, thing)

	return err
}

func (m *ThingModel) afterSave(thing *model.Thing) error {

	conn := m.pool.Get()
	defer conn.Close()

	m.log.Debugf("afterSave - thing received id:%s with device:%s", thing.ID, *thing.DeviceID)

	existingDeviceID, err := m.getDeviceIDForThing(thing.ID)

	if err != nil && err != RecordNotFound {
		return fmt.Errorf("Failed to get existing device relationship error:%s", err)
	}

	if existingDeviceID != thing.DeviceID {
		if thing.DeviceID == nil {

			// Theres no device, so remove the existing relationship if it's there
			deviceID, err := m.getDeviceIDForThing(thing.ID)

			if err == nil {
				err = m.deleteRelationshipWithDevice(*deviceID)
			}

			if err != nil {
				return fmt.Errorf("Failed to remove existing device relationship error:%s", err)
			}

		} else {
			// See if another thing is already attached to the device
			existingThingID, err := m.getThingIDForDevice(thing.ID)

			if existingThingID != nil {
				// Remove the existing relationship
				err = m.deleteRelationshipWithDevice(*thing.DeviceID)

				if err != nil {
					return fmt.Errorf("Failed to remove existing relationship to device %s. Currently attached to thing %s, we wanted it to be attached to %s. Error:%s", *thing.DeviceID, *existingThingID, thing.ID, err)
				}
			}

			_, err = conn.Do("HSET", "device-thing", *thing.DeviceID, thing.ID)

			if err != nil {
				return fmt.Errorf("Failed to update device relationship. error: %s", err)
			}
		}

		if err == nil {
			err = m.markUpdated(thing.ID, time.Now())
			if err != nil {
				return fmt.Errorf("Failed to mark thing updated. error: %s", err)
			}
		}

	}

	return nil
}

func (m *ThingModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	return m.delete(id)
}

func (m *ThingModel) beforeDelete(id string) error {

	// TODO: announce deletion via MQTT
	// self.bus.publish(Ninja.topics.thing.goodbye.thing(thing.id), {id: thing.id});

	conn := m.pool.Get()
	defer conn.Close()

	deviceID, err := m.getDeviceIDForThing(id)

	if err == nil {
		err = m.deleteRelationshipWithDevice(*deviceID)
	}

	return err
}

func (m *ThingModel) FetchByDeviceId(deviceID string) (*model.Thing, error) {
	m.syncing.Wait()

	conn := m.pool.Get()
	defer conn.Close()

	thingID, err := m.getThingIDForDevice(deviceID)

	if err != nil {
		return nil, err
	}

	return m.Fetch(*thingID)
}

func (m *ThingModel) SetLocation(thingID string, roomID *string) error {
	m.syncing.Wait()
	//defer m.sync()

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
	m.syncing.Wait()
	thing := &model.Thing{}

	if err := m.fetch(id, thing); err != nil {
		return nil, fmt.Errorf("Failed to fetch thing (id:%s): %s", id, err)
	}

	return thing, nil
}

func (m *ThingModel) onFetch(thing *model.Thing) error {
	deviceID, err := m.getDeviceIDForThing(thing.ID)

	if err != nil && err != RecordNotFound {
		return fmt.Errorf("Failed to fetch device id for thing (id:%s) : %s", thing.ID, err)
	}

	if deviceID != nil {
		device, err := deviceModel.Fetch(*deviceID)
		if err != nil {
			return fmt.Errorf("Failed to fetch nested device (id:%s) : %s", *deviceID, err)
		}
		thing.Device = device
	}

	return nil
}

func (m *ThingModel) FetchByType(thingType string) (*[]*model.Thing, error) {
	m.syncing.Wait()
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
	m.syncing.Wait()

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
	m.syncing.Wait()
	//defer m.sync()

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

// -- Device<->Thing one-to-one relationship --

func (m *ThingModel) deleteRelationshipWithDevice(deviceID string) error {

	conn := m.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HDEL", "device-thing", deviceID)

	return err
}

func (m *ThingModel) getThingIDForDevice(deviceID string) (*string, error) {
	conn := m.pool.Get()
	defer conn.Close()

	item, err := conn.Do("HGET", "device-thing", deviceID)

	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, RecordNotFound
	}

	thingID, err := redis.String(item, err)

	return &thingID, err
}

func (m *ThingModel) getDeviceIDForThing(thingID string) (*string, error) {
	conn := m.pool.Get()
	defer conn.Close()

	allRels, err := redis.Strings(redis.Values(conn.Do("HGETALL", "device-thing")))

	if err != nil {
		return nil, err
	}

	if len(allRels) == 0 {
		return nil, RecordNotFound
	}

	for i := 0; i < len(allRels); i += 2 {
		if allRels[i+1] == thingID {
			return &allRels[i], nil
		}
	}

	return nil, RecordNotFound
}
