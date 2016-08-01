package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/state"
)

type ThingModel struct {
	baseModel

	DeviceModel  *DeviceModel       `inject:""`
	RoomModel    *RoomModel         `inject:""`
	StateManager state.StateManager `inject:""`
}

var autoPromote = config.Bool(false, "homecloud.autopromote")

func toThing(obj interface{}) *model.Thing {
	var thing, ok = obj.(*model.Thing)
	if !ok {
		panic("Non-'Thing' passed to a ThingModel handler")
	}
	return thing
}

func NewThingModel() *ThingModel {

	thingModel := &ThingModel{
		baseModel: newBaseModel("thing", model.Thing{}),
	}

	thingModel.baseModel.afterSave = func(obj interface{}, conn redis.Conn) error {
		return thingModel.afterSave(toThing(obj), conn)
	}
	thingModel.baseModel.afterDelete = func(obj interface{}, conn redis.Conn) error {
		return thingModel.afterDelete(toThing(obj), conn)
	}
	thingModel.baseModel.onFetch = func(obj interface{}, syncing bool, conn redis.Conn) error {
		return thingModel.onFetch(toThing(obj), syncing, conn)
	}

	return thingModel
}

func (m *ThingModel) ensureThingForDevice(device *model.Device, conn redis.Conn) error {

	// It was a new device, we might need to make a thing for it
	thing, err := m.FetchByDeviceId(device.ID, conn)

	if err != nil && err != RecordNotFound {
		// An actual error
		return err
	}

	if thing != nil {
		// A thing already exists
		return nil
	}

	m.log.Debugf("Device %s has no thing. Creating one", device.ID)

	thing = &model.Thing{
		DeviceID: &device.ID,
		Name:     "New Thing",
		Type:     "unknown",
	}

	if device.Signatures != nil {
		thingType, hasThingType := (*device.Signatures)["ninja:thingType"]

		if hasThingType {
			thing.Type = thingType
			if device.Name == nil {
				thing.Name = "New " + thingType
			}
		}
	}

	// When creating the Thing for a node (like a spheramid) we don't generate a random ID,
	// but instead use the natural ID (the serial number).
	if device.NaturalIDType == "node" {
		thing.ID = device.NaturalID
	}

	if device.Name != nil {
		thing.Name = *device.Name
	}

	if autoPromote {
		thing.Promoted = true
	}

	if err = m.Create(thing, conn); err == nil {
		if autoPromote {
			if id, err := m.RoomModel.ensureDefaultRoom(conn); err == nil {
				err = m.SetLocation(thing.ID, &id, conn)
			}
		}
	}
	return err
}

func (m *ThingModel) Create(thing *model.Thing, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()

	if thing.ID == "" {
		if uuid, err := uuid.NewRandom(); err != nil {
			return err
		} else {
			thing.ID = uuid.String()
		}
	}

	_, err := m.save(thing.ID, thing, conn)

	return err
}

func (m *ThingModel) afterSave(thing *model.Thing, conn redis.Conn) error {

	defer syncFS()

	m.log.Debugf("afterSave - thing received id:%s with device:%s", thing.ID, thing.DeviceID)

	existingDeviceID, err := m.GetDeviceIDForThing(thing.ID, conn)

	if err != nil && err != RecordNotFound {
		return fmt.Errorf("Failed to get existing device relationship error:%s", err)
	}

	if existingDeviceID != thing.DeviceID {
		if thing.DeviceID == nil {

			// Theres no device, so remove the existing relationship if it's there
			deviceID, err := m.GetDeviceIDForThing(thing.ID, conn)

			if err == nil {
				err = m.deleteRelationshipWithDevice(*deviceID, conn)
			}

			if err != nil {
				return fmt.Errorf("Failed to remove existing device relationship error:%s", err)
			}

		} else {
			// See if another thing is already attached to the device
			existingThingID, err := m.GetThingIDForDevice(thing.ID, conn)

			if existingThingID != nil {
				// Remove the existing relationship
				err = m.deleteRelationshipWithDevice(*thing.DeviceID, conn)

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
			err = m.markUpdated(thing.ID, time.Now(), conn)
			if err != nil {
				return fmt.Errorf("Failed to mark thing updated. error: %s", err)
			}
		}

	}

	return nil
}

type DeleteRequest struct {
	ThingID      string `json:"thingID"`
	DeleteDevice bool   `json:"deleteDevice"`
}

func (m *ThingModel) Delete(r *DeleteRequest, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()
	id := r.ThingID
	deleteDevice := r.DeleteDevice

	if deleteDevice {
		deviceID, err := m.GetDeviceIDForThing(id, conn)

		if err == nil && deviceID != nil {
			m.deleteRelationshipWithDevice(*deviceID, conn)
			err = m.DeviceModel.Delete(*deviceID, conn)
			if err != nil {
				m.log.Infof("Failed to delete attached device: %s when removing thing: %s. Continuing. error:%s", deviceID, id, err)
			}
		}
	}

	return m.delete(id, conn)
}

func (m *ThingModel) afterDelete(deletedThing *model.Thing, conn redis.Conn) error {

	// TODO: announce deletion via MQTT
	// self.bus.publish(Ninja.topics.thing.goodbye.thing(thing.id), {id: thing.id});

	deviceID, err := m.GetDeviceIDForThing(deletedThing.ID, conn)

	if err == nil {
		err = m.deleteRelationshipWithDevice(*deviceID, conn)
	}

	if err == RecordNotFound {
		// The device the deleted thing was attached to no longer exists anyway
		return nil
	}

	device, err := m.DeviceModel.Fetch(*deviceID, conn)

	if err != nil && err != RecordNotFound {
		return err
	}

	// Create a new, unpromoted thing for the device
	return m.ensureThingForDevice(device, conn)
}

func (m *ThingModel) FetchByDeviceId(deviceID string, conn redis.Conn) (*model.Thing, error) {
	m.syncing.Wait()

	thingID, err := m.GetThingIDForDevice(deviceID, conn)

	if err != nil {
		return nil, err
	}

	return m.Fetch(*thingID, conn)
}

func (m *ThingModel) SetLocation(thingID string, roomID *string, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()

	existing, err := m.Fetch(thingID, conn)

	if err != nil {
		return err
	}

	if existing.Location == roomID {
		// Nothing to do
		return nil
	}

	err = m.RoomModel.MoveThing(existing.Location, roomID, thingID, conn)

	if err != nil {
		if roomID == nil {
			return fmt.Errorf("Failed to remove thing %s from %s", thingID, existing.Location)
		}

		return fmt.Errorf("Failed to move thing %s from %s to %s", thingID, existing.Location, *roomID)
	}

	existing.Location = roomID

	// TODO: NOT HANDLING TAGS ETC!!!!
	// If we are moving it into no room, unpromote the device
	if roomID == nil {
		existing.Promoted = false
	}

	_, err = m.save(thingID, existing, conn)
	return err
}

func (m *ThingModel) Fetch(id string, conn redis.Conn) (*model.Thing, error) {
	m.syncing.Wait()
	thing := &model.Thing{}

	if err := m.fetch(id, thing, false, conn); err != nil {
		return nil, fmt.Errorf("Failed to fetch thing (id:%s): %s", id, err)
	}

	m.StateManager.Merge(thing)

	return thing, nil
}

func (m *ThingModel) onFetch(thing *model.Thing, syncing bool, conn redis.Conn) error {
	deviceID, err := m.GetDeviceIDForThing(thing.ID, conn)

	if err != nil && err != RecordNotFound {
		return fmt.Errorf("Failed to fetch device id for thing (id:%s) : %s", thing.ID, err)
	}

	if deviceID != nil {

		thing.DeviceID = deviceID

		if !syncing {
			device, err := m.DeviceModel.Fetch(*deviceID, conn)
			if err != nil {
				return fmt.Errorf("Failed to fetch nested device (id:%s) on thing %s : %s", *deviceID, thing.ID, err)
			}
			thing.Device = device
		}
	}

	return nil
}

func (m *ThingModel) FetchByType(thingType string, conn redis.Conn) (*[]*model.Thing, error) {
	m.syncing.Wait()
	allThings, err := m.FetchAll(conn)

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

func (m *ThingModel) FetchAll(conn redis.Conn) (*[]*model.Thing, error) {
	m.syncing.Wait()

	ids, err := m.fetchIds(conn)

	if err != nil {
		return nil, err
	}

	things := make([]*model.Thing, len(ids))

	for i, id := range ids {
		things[i], err = m.Fetch(id, conn)
		if err != nil {
			return nil, err
		}
	}

	return &things, nil
}

// Update a thing, this is currently very optimisic and only changes name and type fields.
func (m *ThingModel) Update(id string, thing *model.Thing, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()

	oldThing := &model.Thing{}

	if err := m.fetch(id, oldThing, false, conn); err != nil {
		return fmt.Errorf("Failed to fetch thing (id:%s): %s", id, err)
	}

	oldThing.Name = thing.Name
	oldThing.Type = thing.Type
	oldThing.Promoted = thing.Promoted

	if _, err := m.save(id, oldThing, conn); err != nil {
		return fmt.Errorf("Failed to update thing (id:%s): %s", id, err)
	}

	return nil
}

// -- Device<->Thing one-to-one relationship --

func (m *ThingModel) deleteRelationshipWithDevice(deviceID string, conn redis.Conn) error {

	defer syncFS()

	_, err := conn.Do("HDEL", "device-thing", deviceID)

	return err
}

func (m *ThingModel) GetThingIDForDevice(deviceID string, conn redis.Conn) (*string, error) {

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

func (m *ThingModel) GetDeviceIDForThing(thingID string, conn redis.Conn) (*string, error) {

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
