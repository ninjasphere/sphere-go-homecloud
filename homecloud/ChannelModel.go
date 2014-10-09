package homecloud

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type ChannelModel struct {
	baseModel
}

func NewChannelModel(pool *redis.Pool, conn *ninja.Connection) *ChannelModel {
	return &ChannelModel{
		baseModel{
			syncing: &sync.WaitGroup{},
			pool:    pool,
			idType:  "channel",
			objType: reflect.TypeOf(model.Channel{}),
			conn:    conn,
			log:     logger.GetLogger("ChannelModel"),
		},
	}
}

func (m *ChannelModel) Create(deviceID string, channel *model.Channel) error {
	m.syncing.Wait()
	//defer m.sync()

	channel.DeviceID = deviceID

	if _, err := m.save(deviceID+"-"+channel.ID, channel); err != nil {
		return err
	}

	conn := m.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("SADD", "device:"+deviceID+":channels", channel.ID); err != nil {
		return err
	}

	return nil
}

func (m *ChannelModel) Delete(deviceID string, channelID string) error {
	m.syncing.Wait()
	//defer m.sync()

	err := m.delete(deviceID + "-" + channelID)
	if err != nil {
		return err
	}

	conn := m.pool.Get()
	defer conn.Close()

	_, err = conn.Do("SREM", "device:"+deviceID+":channels", channelID)

	// TODO: announce deletion via MQTT
	// publish(Ninja.topics.room.goodbye.room(roomId)
	// publish(Ninja.topics.location.calibration.delete, {zone: roomId})

	return err
}

func (m *ChannelModel) FetchAll(deviceID string) (*[]*model.Channel, error) {
	m.syncing.Wait()

	conn := m.pool.Get()
	defer conn.Close()
	ids, err := redis.Strings(conn.Do("SMEMBERS", "device:"+deviceID+":channels"))
	m.log.Debugf("Found %d channel id(s) for device %s", len(ids), deviceID)

	if err != nil {
		return nil, err
	}

	channels := make([]*model.Channel, len(ids))

	for i, id := range ids {
		channels[i], err = m.Fetch(deviceID, id)
		if err != nil {
			return nil, err
		}
	}

	return &channels, nil
}

func (m *ChannelModel) Fetch(deviceID, channelID string) (*model.Channel, error) {
	m.syncing.Wait()
	channel := &model.Channel{}

	if err := m.fetch(deviceID+"-"+channelID, channel, false); err != nil {
		return nil, fmt.Errorf("Failed to fetch channel (device id: %s channel id:%s): %s", deviceID, channelID, err)
	}

	return channel, nil
}
