package models

func GetInjectables() []interface{} {
	// Oh i wish inject worked with nested

	moduleModel := NewModuleModel()
	channelModel := NewChannelModel()
	deviceModel := NewDeviceModel()
	roomModel := NewRoomModel()
	siteModel := NewSiteModel()
	thingModel := NewThingModel()

	return []interface{}{
		moduleModel, &moduleModel.baseModel,
		channelModel, &channelModel.baseModel,
		deviceModel, &deviceModel.baseModel,
		roomModel, &roomModel.baseModel,
		siteModel, &siteModel.baseModel,
		thingModel, &thingModel.baseModel,
	}
}
