package ts

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

var log = logger.GetLogger("ts")

type TimeSeriesManager struct {
	Conn         *ninja.Connection    `inject:""`
	ThingModel   *models.ThingModel   `inject:""`
	ChannelModel *models.ChannelModel `inject:""`
	Pool         *redis.Pool          `inject:""`
	outgoing     chan *TimeSeriesPayload
	log          *logger.Logger
}

type TimeSeriesPayload struct {
	Thing      string                        `json:"thing"`
	ThingType  string                        `json:"thingType"`
	Promoted   bool                          `json:"promoted"`
	Device     string                        `json:"device"`
	Channel    string                        `json:"channel"`
	Schema     string                        `json:"schema"`
	Event      string                        `json:"event"`
	Points     []schemas.TimeSeriesDatapoint `json:"points"`
	Time       int64                         `json:"time"`
	TimeZone   string                        `json:"timeZone"`
	TimeOffset int                           `json:"timeOffset"`
	Site       string                        `json:"site"`
	_User      string                        `json:"_"`
}

func (m *TimeSeriesManager) PostConstruct() error {
	m.log = logger.GetLogger("TimeSeriesManager")
	m.outgoing = make(chan *TimeSeriesPayload, 1)

	if config.Bool(false, "homecloud.influx.enable") {
		influx, err := newinfluxRecorder()

		if err != nil {
			return err
		}

		go influx.messageHandler(m.outgoing)
	}

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

	t := Tick{
		name: "TimeSeries per/sec",
	}
	t.start()

	influxEnabled := config.Bool(false, "homecloud.influx.enable")

	m.Conn.GetMqttClient().Subscribe("$device/+/channel/+/event/state", func(topic string, message []byte) {
		t.tick()

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

			payload := &TimeSeriesPayload{
				Thing:     thing.ID,
				ThingType: thing.Type,
				Promoted:  thing.Promoted,
				Device:    values["device"],
				Channel:   values["channel"],
				Schema:    channel.Schema,
				Event:     "state",
				Points:    points,
				Site:      config.MustString("siteId"),
				Time:      int64(data["time"].(float64)),
			}

			if user, ok := data["_userOverride"].(string); ok {
				payload._User = user
			}

			if site, ok := data["_siteOverride"].(string); ok {
				payload.Site = site
			}

			payload.TimeZone, payload.TimeOffset = time.Now().Zone()

			if influxEnabled {
				m.outgoing <- payload
			} else {
				err = m.Conn.SendNotification("$ninja/services/timeseries", payload)
				if err != nil {
					log.Fatalf("Got a state event, but failed to send time series points. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
				}
			}

		}

	})

	return nil
}

type Tick struct {
	count int
	name  string
}

func (t *Tick) tick() {
	t.count++
}

func (t *Tick) start() {
	go func() {
		for {
			time.Sleep(time.Second)
			//spew.Dump(t.name, t.count)
			t.count = 0
		}
	}()
}
