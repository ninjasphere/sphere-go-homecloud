package models

import (
	"fmt"

	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/redigo/redis"
)

type ChannelModel struct {
	baseModel
}

func NewChannelModel() *ChannelModel {
	return &ChannelModel{
		baseModel: newBaseModel("channel", model.Channel{}),
	}

}

func (m *ChannelModel) Create(deviceID string, channel *model.Channel, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()
	defer syncFS()

	channel.DeviceID = deviceID

	if _, err := m.save(deviceID+"-"+channel.ID, channel, conn); err != nil {
		return err
	}

	if _, err := conn.Do("SADD", "device:"+deviceID+":channels", channel.ID); err != nil {
		return err
	}

	return nil
}

func (m *ChannelModel) Delete(deviceID string, channelID string, conn redis.Conn) error {
	m.syncing.Wait()
	//defer m.sync()
	defer syncFS()

	err := m.delete(deviceID+"-"+channelID, conn)
	if err != nil {
		return err
	}

	_, err = conn.Do("SREM", "device:"+deviceID+":channels", channelID)

	// TODO: announce deletion via MQTT
	// publish(Ninja.topics.room.goodbye.room(roomId)
	// publish(Ninja.topics.location.calibration.delete, {zone: roomId})

	return err
}

func (m *ChannelModel) FetchAll(deviceID string, conn redis.Conn) (*[]*model.Channel, error) {
	m.syncing.Wait()

	ids, err := redis.Strings(conn.Do("SMEMBERS", "device:"+deviceID+":channels"))
	m.log.Debugf("Found %d channel id(s) for device %s", len(ids), deviceID)

	if err != nil {
		return nil, err
	}

	channels := make([]*model.Channel, len(ids))

	for i, id := range ids {
		channels[i], err = m.Fetch(deviceID, id, conn)
		if err != nil {
			return nil, err
		}
	}

	return &channels, nil
}

func (m *ChannelModel) Fetch(deviceID, channelID string, conn redis.Conn) (*model.Channel, error) {
	m.syncing.Wait()
	channel := &model.Channel{}

	if err := m.fetch(deviceID+"-"+channelID, channel, false, conn); err != nil {
		return nil, fmt.Errorf("Failed to fetch channel (device id: %s channel id:%s): %s", deviceID, channelID, err)
	}

	return channel, nil
}
