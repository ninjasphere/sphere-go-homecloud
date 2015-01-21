package models

import (
	"os/exec"
	"time"

	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
)

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

var fsSyncLog = logger.GetLogger("FS Sync")
var fsSyncInterval = config.Duration(time.Second, "homecloud.fsSync.minInterval")
var fsSyncEnabled = config.Bool(false, "homecloud.fsSync.enabled")
var fsSyncTask *time.Timer

func syncFS() {
	if fsSyncEnabled {
		if fsSyncTask == nil {
			fsSyncTask = time.AfterFunc(fsSyncInterval, func() {
				fsSyncLog.Debugf("Syncing filesystem...")
				exec.Command("sync").Output()
			})
		}

		fsSyncTask.Reset(fsSyncInterval)
	}
}
