package models

func GetInjectables() []interface{} {
	// Oh i wish inject worked with nested

	appModel := NewAppModel()
	channelModel := NewChannelModel()
	deviceModel := NewDeviceModel()
	driverModel := NewDriverModel()
	roomModel := NewRoomModel()
	siteModel := NewSiteModel()
	thingModel := NewThingModel()

	return []interface{}{
		appModel, &appModel.baseModel,
		channelModel, &channelModel.baseModel,
		deviceModel, &deviceModel.baseModel,
		driverModel, &driverModel.baseModel,
		roomModel, &roomModel.baseModel,
		siteModel, &siteModel.baseModel,
		thingModel, &thingModel.baseModel,
	}
}
