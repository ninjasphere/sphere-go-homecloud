package models

import (
	"fmt"

	"github.com/ninjasphere/go-ninja/model"
)

type DeviceModel struct {
	baseModel

	Things   *ThingModel   `inject:""`
	Channels *ChannelModel `inject:""`
}

func NewDeviceModel() *DeviceModel {
	return &DeviceModel{
		baseModel: newBaseModel("device", model.Device{}),
	}
}

func (m *DeviceModel) Fetch(deviceID string) (*model.Device, error) {
	m.syncing.Wait()

	device := &model.Device{}

	if err := m.fetch(deviceID, device, false); err != nil {
		return nil, err
	}

	channels, err := m.Channels.FetchAll(deviceID)

	if err != nil {
		return nil, fmt.Errorf("Failed to get channels for device id:%s error:%s", deviceID, err)
	}

	device.Channels = channels

	thingID, err := m.Things.getThingIDForDevice(deviceID)

	if err != nil && err != RecordNotFound {
		return nil, err
	}

	device.ThingID = thingID

	return device, nil
}

func (m *DeviceModel) FetchAll() (*[]*model.Device, error) {
	m.syncing.Wait()

	ids, err := m.fetchIds()

	if err != nil {
		return nil, err
	}

	devices := make([]*model.Device, len(ids))

	for i, id := range ids {
		devices[i], err = m.Fetch(id)
		if err != nil {
			return nil, err
		}
	}

	return &devices, nil
}

func (m *DeviceModel) Create(device *model.Device) error {
	m.syncing.Wait()
	//defer m.sync()

	m.log.Debugf("Saving device %s", device.ID)

	updated, err := m.save(device.ID, device)

	m.log.Debugf("Device was updated? %t", updated)

	if err != nil {
		return err
	}

	return m.Things.ensureThingForDevice(device)
}

func (m *DeviceModel) Delete(id string) error {
	m.syncing.Wait()
	//defer m.sync()

	err := m.delete(id)
	if err != nil {
		err = m.Things.deleteRelationshipWithDevice(id)
	}
	return err
}
