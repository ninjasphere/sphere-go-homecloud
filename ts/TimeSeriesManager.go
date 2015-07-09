package ts

import (
	"encoding/json"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/schemas"
	"github.com/ninjasphere/sphere-go-homecloud/models"
)

var log = logger.GetLogger("ts")

type TimeSeriesManager struct {
	Conn         *ninja.Connection    `inject:""`
	ThingModel   *models.ThingModel   `inject:""`
	ChannelModel *models.ChannelModel `inject:""`
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

func (m *TimeSeriesManager) Start() error {

	m.log.Infof("Starting")

	_, err := m.Conn.GetServiceClient("$device/:device/channel/:channel").OnEvent("state", func(params *json.RawMessage, values map[string]string) bool {

		thing, err := m.ThingModel.FetchByDeviceId(values["device"])
		if err != nil {
			log.Errorf("Got a state event, but failed to fetch thing for device: %s error: %s", values["device"], err)
			return true
		}

		if thing == nil {
			return true
		}

		channel, err := m.ChannelModel.Fetch(values["device"], values["channel"])

		if err != nil {
			log.Errorf("Got a state event, but failed to fetch channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			return true
		}

		var data interface{}

		err = json.Unmarshal(*params, &data)

		if err != nil {
			log.Errorf("Got a state event, but failed to unmarshal it. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			return true
		}

		log.Debugf("Got state event from device:%s channel:%s payload:%v", values["device"], values["channel"], data)

		points, err := schemas.GetEventTimeSeriesData(data, channel.Schema, "state")
		if err != nil {
			log.Errorf("Got a state event, but failed to create time series points. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			return true
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
				Time:      time.Now().UnixNano() / int64(time.Millisecond),
				Site:      config.MustString("siteId"),
			}

			payload.TimeZone, payload.TimeOffset = time.Now().Zone()

			err = m.Conn.SendNotification("$ninja/services/timeseries", payload)
			if err != nil {
				log.Fatalf("Got a state event, but failed to send time series points. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			}

			select {
			case m.outgoing <- payload:
			default:
			}

		}

		return true
	})

	return err
}
