package homecloud

import (
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type RoomModel struct {
	conn redis.Conn
	log  *logger.Logger
}

func NewRoomModel(conn redis.Conn) *RoomModel {
	return &RoomModel{
		conn: conn,
		log:  logger.GetLogger("RoomModel"),
	}
}

func (m *RoomModel) Fetch(id string) (*model.Room, error) {

	item, err := redis.Values(m.conn.Do("HGETALL", "Room:"+id))

	if err != nil {
		return nil, err
	}

	if len(item) == 0 {
		return nil, nil
	}

	Room := &model.Room{}

	if err := redis.ScanStruct(item, Room); err != nil {
		return nil, err
	}

	return Room, nil
}

func (m *RoomModel) MoveThing(from *string, to *string, thing string) error {
	var err error

	if to != nil && from == nil {
		// we need to add it

		dest := "room:" + *to + ":things"

		m.log.Debugf("Adding thing %s to room %s", thing, *to)
		_, err = m.conn.Do("SADD", dest, thing)

	} else if from != nil && to == nil {
		// we need to remove it

		src := "room:" + *from + "s:things"

		m.log.Debugf("Removing thing %s from room %s", thing, *from)
		_, err = m.conn.Do("SREM", src, thing)

	} else {
		// need to move it

		src := "room:" + *from + "s:things"
		dest := "room:" + *to + ":things"

		m.log.Debugf("Moving thing %s from room %s to room %s", thing, *from, *to)

		result, err2 := m.conn.Do("SMOVE", src, dest, thing)
		if err2 != nil {
			return err2
		}

		if result == 0 {
			// Wasn"t in src, we should add it to dest
			_, err = m.conn.Do("SADD", dest, thing)
		}

	}

	return err

}
