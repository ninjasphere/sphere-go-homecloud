package homecloud

import (
	"encoding/json"
	"time"

	"github.com/ninjasphere/go-ninja/schemas"
)

type timeSeriesPayload struct {
	Thing      string                        `json:"thing"`
	ThingType  string                        `json:"thingType"`
	Device     string                        `json:"device"`
	Channel    string                        `json:"channel"`
	Schema     string                        `json:"schema"`
	Event      string                        `json:"event"`
	Points     []schemas.TimeSeriesDatapoint `json:"points"`
	Time       int64                         `json:"time"`
	TimeZone   string                        `json:"timeZone"`
	TimeOffset int                           `json:"timeOffset"`
}

func startManagingTimeSeries() {
	err := conn.GetServiceClient("$device/:device/channel/:channel").OnEvent("state", func(params *json.RawMessage, values map[string]string) bool {

		thing, err := thingModel.FetchByDeviceId(values["device"])
		if err != nil {
			log.Errorf("Got a state event, but failed to fetch thing for device: %s error: %s", values["device"], err)
			return true
		}

		if thing == nil {
			return true
		}

		channel, err := channelModel.Fetch(values["device"], values["channel"])

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

			payload := &timeSeriesPayload{
				Thing:     thing.ID,
				ThingType: thing.Type,
				Device:    values["device"],
				Channel:   values["channel"],
				Schema:    channel.Schema,
				Event:     "state",
				Points:    points,
				Time:      time.Now().UnixNano() / int64(time.Millisecond),
			}

			payload.TimeZone, payload.TimeOffset = time.Now().Zone()

			err = conn.SendNotification("$ninja/services/timeseries", payload)
			if err != nil {
				log.Fatalf("Got a state event, but failed to send time series points. channel: %s on device: %s error: %s", values["channel"], values["device"], err)
			}
		}

		return true
	})

	if err != nil {
		log.FatalError(err, "Failed to register for state events in the time series manager.")
	}
}
