package homecloud

import (
	"log"

	"github.com/davecgh/go-spew/spew"

	"github.com/ninjasphere/redigo/redis"
)

var thingModel *ThingModel
var deviceModel *DeviceModel

func StartHomeCloud() {
	c, err := redis.Dial("tcp", ":6379")

	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	thingModel = NewThingModel(c)

	deviceModel = NewDeviceModel(c)

	spew.Dump(thingModel.FetchByDeviceId("13294d4619"))

}
