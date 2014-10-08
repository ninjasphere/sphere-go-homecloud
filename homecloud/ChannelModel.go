package homecloud

import (
	"fmt"
	"reflect"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type ChannelModel struct {
	baseModel
}

func NewChannelModel(pool *redis.Pool, conn *ninja.Connection) *ChannelModel {
	return &ChannelModel{baseModel{pool, "channel", reflect.TypeOf(model.Channel{}), conn, logger.GetLogger("ChannelModel")}}
}

func (m *ChannelModel) MustSync() {
	if err := m.sync(); err != nil {
		m.log.Fatalf("Failed to sync channels error:%s", err)
	}
}

func (m *ChannelModel) Create(deviceID string, channel *model.Channel) error {
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
	channel := &model.Channel{}

	if err := m.fetch(deviceID+"-"+channelID, channel); err != nil {
		return nil, fmt.Errorf("Failed to fetch channel (device id: %s channel id:%s): %s", deviceID, channelID, err)
	}

	return channel, nil
}
