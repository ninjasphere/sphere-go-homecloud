package homecloud

import (
	"fmt"
	"reflect"
	"sync"

	"code.google.com/p/go-uuid/uuid"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type RoomModel struct {
	baseModel
}

func NewRoomModel(pool *redis.Pool, conn *ninja.Connection) *RoomModel {
	return &RoomModel{
		baseModel{
			syncing: &sync.WaitGroup{},
			pool:    pool,
			idType:  "room",
			objType: reflect.TypeOf(model.Room{}),
			conn:    conn,
			log:     logger.GetLogger("RoomModel"),
		},
	}
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

	err := m.delete(id)
	if err != nil {
		return err
	}

	conn := m.pool.Get()
	defer conn.Close()

	_, err = conn.Do("DEL", fmt.Sprintf("room:%s:things", id))

	// TODO: announce deletion via MQTT
	// publish(Ninja.topics.room.goodbye.room(roomId)
	// publish(Ninja.topics.location.calibration.delete, {zone: roomId})

	return err
}

func (m *RoomModel) MoveThing(from *string, to *string, thing string) error {
	m.syncing.Wait()
	//defer m.sync()

	var err error

	conn := m.pool.Get()
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
