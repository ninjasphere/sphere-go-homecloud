package models

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type RoomModel struct {
	baseModel

	SiteModel  *SiteModel  `inject:""`
	ThingModel *ThingModel `inject:""`
	Pool       *redis.Pool `inject:""`
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
	roomModel.baseModel.afterDelete = func(obj interface{}, conn redis.Conn) error {
		return roomModel.afterDelete(toRoom(obj), conn)
	}

	return roomModel
}

func (m *RoomModel) PostConstruct() error {

	log.Printf("Checking for bad rooms...")

	conn := m.Pool.Get()
	defer conn.Close()

	// Check all rooms to see if we have any without an ID.
	// Delete any that are missing an ID.
	ids, err := m.fetchIds(conn)

	if err != nil {
		return err
	}

	for _, id := range ids {
		room, err := m.Fetch(id, conn)
		if err != nil || !m.isValid(room) {
			log.Printf("Bad room! ID: %s Room: %+v", id, room)
			m.Delete(id, conn)
		}
	}

	return nil
}

// Do more here. Use the schemas.
func (m *RoomModel) isValid(room *model.Room) bool {
	return room.ID != ""
}

func (m *RoomModel) Create(room *model.Room, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()

	if room.ID == "" {
		if uuid, err := uuid.NewRandom(); err != nil {
			return err
		} else {
			room.ID = uuid.String()
		}
	}

	_, err := m.save(room.ID, room, conn)
	return err
}

func (m *RoomModel) Fetch(id string, conn redis.Conn) (*model.Room, error) {
	m.syncing.Wait()

	room := &model.Room{}

	if err := m.fetch(id, room, false, conn); err != nil {
		return nil, err
	}

	return room, nil
}

func (m *RoomModel) FetchAll(conn redis.Conn) (*[]*model.Room, error) {
	m.syncing.Wait()

	ids, err := m.fetchIds(conn)

	if err != nil {
		return nil, err
	}

	rooms := make([]*model.Room, len(ids))

	for i, id := range ids {
		rooms[i], err = m.Fetch(id, conn)
		if err != nil {
			return nil, err
		}
	}

	return &rooms, nil
}

func (m *RoomModel) Delete(id string, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()

	return m.delete(id, conn)
}

func (m *RoomModel) afterDelete(deletedRoom *model.Room, conn redis.Conn) error {

	defer syncFS()

	thingIds, err := redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("room:%s:things", deletedRoom.ID)))

	for _, id := range thingIds {

		err := m.ThingModel.SetLocation(id, nil, conn)

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

func (m *RoomModel) MoveThing(from *string, to *string, thing string, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()

	var err error

	if to != nil {
		_, err := m.Fetch(*to, conn) // Ensure the room we are putting it into actually exists

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

//
// This procedure checks that that the current site has a default room
// and, if not, creates one then updates the site to record the identity
// of the created room.
//
func (m *RoomModel) ensureDefaultRoom(conn redis.Conn) (string, error) {
	if site, err := m.SiteModel.Fetch("here", conn); err != nil {
		return "", err
	} else {
		var room *model.Room
		var roomID string
		var err error

		if site.DefaultRoomID != nil && *site.DefaultRoomID != "" {
			roomID = *site.DefaultRoomID
			room, err = m.Fetch(roomID, conn)
		}

		if room != nil && err == nil {
			return room.ID, nil
		} else {

			room = &model.Room{
				Name: "Default Room for Site",
				Type: "default",
			}

			if err := m.Create(room, conn); err != nil {
				log.Println("failed to create room: %v", err)
				return "", err
			}

			site.DefaultRoomID = &room.ID
			log.Println("created default room id: %s", room.ID)

			if err := m.SiteModel.Update(site.ID, site, conn); err != nil {
				log.Println("failed to update site: %v", err)
				return "", err
			} else {
				return room.ID, nil
			}
		}
	}
}
