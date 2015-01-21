package models

import (
	"fmt"

	"code.google.com/p/go-uuid/uuid"

	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type RoomModel struct {
	baseModel

	ThingModel *ThingModel `inject:""`
}

func toRoom(obj interface{}) *model.Room {
	var thing, ok = obj.(*model.Room)
	if !ok {
		panic("Non-'Room' passed to a RoomModel handler")
	}
	return thing
}

func NewRoomModel() *RoomModel {

	roomModel := &RoomModel{
		baseModel: newBaseModel("room", model.Room{}),
	}
	roomModel.baseModel.afterDelete = func(obj interface{}) error {
		return roomModel.afterDelete(toRoom(obj))
	}

	return roomModel
}

func (m *RoomModel) Create(room *model.Room) error {
	m.syncing.Wait()
	//defer m.sync()

	if room.ID == "" {
		room.ID = uuid.NewUUID().String()
	}

	_, err := m.save(room.ID, room)
	return err
}

func (m *RoomModel) Fetch(id string) (*model.Room, error) {
	m.syncing.Wait()

	room := &model.Room{}

	if err := m.fetch(id, room, false); err != nil {
		return nil, err
	}

	return room, nil
}

func (m *RoomModel) FetchAll() (*[]*model.Room, error) {
	m.syncing.Wait()

	ids, err := m.fetchIds()

	if err != nil {
		return nil, err
	}

	rooms := make([]*model.Room, len(ids))

	for i, id := range ids {
		rooms[i], err = m.Fetch(id)
		if err != nil {
			return nil, err
		}
	}

	return &rooms, nil
}

func (m *RoomModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	return m.delete(id)
}

func (m *RoomModel) afterDelete(deletedRoom *model.Room) error {

	conn := m.Pool.Get()
	defer conn.Close()
	defer syncFS()

	thingIds, err := redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("room:%s:things", deletedRoom.ID)))

	for _, id := range thingIds {

		err := m.ThingModel.SetLocation(id, nil)

		if err == RecordNotFound {
			// We were out of sync, but don't really care...
			continue
		}
		if err != nil {
			m.log.Infof("Failed to fetch thing that was in a deleted room. ID: %s error: %s", id, err)
		}

	}

	_, err = conn.Do("DEL", fmt.Sprintf("room:%s:things", deletedRoom.ID))

	return err
}

func (m *RoomModel) MoveThing(from *string, to *string, thing string) error {
	m.syncing.Wait()
	//defer m.sync()

	var err error

	conn := m.Pool.Get()
	defer conn.Close()

	if to != nil {
		_, err := m.Fetch(*to) // Ensure the room we are putting it into actually exists

		if err != nil {
			return err
		}
	}

	// Don't do a damn thing.
	if from == to {
		return nil
	}

	defer syncFS()

	if to != nil && from == nil {
		// we need to add it

		dest := "room:" + *to + ":things"

		m.log.Debugf("Adding thing %s to room %s", thing, *to)
		_, err = conn.Do("SADD", dest, thing)

	} else if from != nil && to == nil {
		// we need to remove it

		src := "room:" + *from + "s:things"

		m.log.Debugf("Removing thing %s from room %s", thing, *from)
		_, err = conn.Do("SREM", src, thing)

	} else {
		// need to move it

		src := "room:" + *from + "s:things"
		dest := "room:" + *to + ":things"

		m.log.Debugf("Moving thing %s from room %s to room %s", thing, *from, *to)

		result, err2 := conn.Do("SMOVE", src, dest, thing)
		if err2 != nil {
			return err2
		}

		if result == 0 {
			// Wasn"t in src, we should add it to dest
			_, err = conn.Do("SADD", dest, thing)
		}

	}

	return err

}
