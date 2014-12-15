package models

// SyncFromCloud pull down changes from cloud, ATM this is DISABLED because of bugs and issues surrounding deletions.
const SyncFromCloud = false

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
