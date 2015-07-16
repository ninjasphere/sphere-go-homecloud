package homecloud

import (
	"encoding/json"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-ninja/schemas"
	"github.com/ninjasphere/redigo/redis"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

type TimeSeriesManager struct {
	Conn         *ninja.Connection    `inject:""`
	ThingModel   *models.ThingModel   `inject:""`
	ChannelModel *models.ChannelModel `inject:""`
	Pool         *redis.Pool          `inject:""`
	outgoing     chan *model.TimeSeriesPayload
	log          *logger.Logger
}

func (m *TimeSeriesManager) PostConstruct() error {
	m.log = logger.GetLogger("TimeSeriesManager")
	m.outgoing = make(chan *model.TimeSeriesPayload, 1)

	return m.Start()
}

var thingsByDeviceId = map[string]*model.Thing{}
var channels = map[string]*model.Channel{}

func init() {
	go func() {
		for {
			time.Sleep(time.Second)
			thingsByDeviceId = map[string]*model.Thing{}
			channels = map[string]*model.Channel{}
		}
	}()
}

func (m *TimeSeriesManager) Start() error {

	m.log.Infof("Starting")

	m.Conn.GetMqttClient().Subscribe("$device/+/channel/+/event/state", func(topic string, message []byte) {

		x, _ := ninja.MatchTopicPattern("$device/:device/channel/:channel/event/state", topic)
		values := *x

		thing, inCache := thingsByDeviceId[values["device"]]

		if !inCache {

			conn := m.Pool.Get()
			defer conn.Close()

			var err error
			thing, err = m.ThingModel.FetchByDeviceId(values["device"], conn)
			if err != nil {
				log.Errorf("Got a state event, but failed to fetch thing for device: %s error: %s", values["device"], err)
				return
			}

			if thing == nil {
				return
			}

			thingsByDeviceId[values["device"]] = thing
		}

		channel, inCache := channels[values["device"]+values["channel"]]

		if !inCache {
			conn := m.Pool.Get()
			defer conn.Close()

			var err error
			channel, err = m.ChannelModel.Fetch(values["device"], values["channel"], conn)

			if err != nil {
				log.Errorf("Got a state event, but failed to fetch channel: %s on device: %s error: %s", values["channel"], values["device"], err)
				return
			}

			channels[values["device"]+values["channel"]] = channel
		}

		var data map[string]interface{}

		err := json.Unmarshal(message, &data)

		params := data["params"]
		if paramsArray, ok := data["params"].([]interface{}); ok {
			params = paramsArray[0]
		}

		if err != nil {
			log.Errorf("Got a state event, but failed to unmarshal it. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			return
		}

		log.Debugf("Got state event from device:%s channel:%s payload:%v", values["device"], values["channel"], params)

		points, err := schemas.GetEventTimeSeriesData(params, channel.Schema, "state")
		if err != nil {
			log.Errorf("Got a state event, but failed to create time series points. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			return
		}

		if len(points) > 0 {

			payload := &model.TimeSeriesPayload{
				Thing:     thing.ID,
				ThingType: thing.Type,
				Promoted:  thing.Promoted,
				Device:    values["device"],
				Channel:   values["channel"],
				Schema:    channel.Schema,
				Event:     "state",
				Points:    points,
				Site:      config.MustString("siteId"),
				Time:      time.Now().Format(time.RFC3339Nano),
			}

			/*if user, ok := data["_userOverride"].(string); ok {
				payload.UserOverride = user
			}

			if node, ok := data["_nodeOverride"].(string); ok {
				payload.NodeOverride = node
			}

			if site, ok := data["_siteOverride"].(string); ok {
				payload.SiteOverride = site
			}*/

			payload.TimeZone, payload.TimeOffset = time.Now().Zone()

			err = m.Conn.SendNotification("$ninja/services/timeseries", payload)
			if err != nil {
				log.Fatalf("Got a state event, but failed to send time series points. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			}

		}

	})

	return nil
}
