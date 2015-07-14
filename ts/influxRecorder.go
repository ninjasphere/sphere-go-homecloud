package ts

import (
	"fmt"
	"net/url"
	"time"

	"github.com/influxdb/influxdb/client"
	"github.com/ninjasphere/go-ninja/config"
)

type InfluxRecorder struct {
	client   *client.Client // influxdb client
	incoming chan *TimeSeriesPayload
}

var tps = Tick{
	name: "TimeSeries per/sec",
}

func NewInfluxRecorder() (*InfluxRecorder, error) {
	host, err := url.Parse(fmt.Sprintf("http://%s:%d", config.String("localhost", "homecloud.influx.host"), 8086))
	if err != nil {
		return nil, err
	}

	influx, err := client.NewClient(client.Config{URL: *host})

	if err != nil {
		return nil, err
	}

	i := &InfluxRecorder{
		client:   influx,
		incoming: make(chan *TimeSeriesPayload),
	}

	go i.messageHandler()

	tps.start()

	return i, nil
}

func (k *InfluxRecorder) Send(p *TimeSeriesPayload) {
	//k.sendTimeseries([]*TimeSeriesPayload{p})
	k.incoming <- p
}

func (k *InfluxRecorder) messageHandler() {

	for p := range k.incoming {

		start := time.Now()

		err := k.sendTimeseries(p)

		if err != nil {
			log.Errorf("failed to post payload: %s", err)
		}

		log.Debugf("response Time Taken: %v", time.Since(start), p)

	}
	log.Infof("handle: deliveries channel closed")

}

func (k *InfluxRecorder) sendTimeseries(t *TimeSeriesPayload) error {

	bps := client.BatchPoints{
		Points:   []client.Point{},
		Database: config.String("sphere", "homecloud.influx.database"),
	}

	tps.tick()

	var key = fmt.Sprintf("%s.%s.%s.%s", t.ThingType, t.Channel, t.Event, t.Site)

	timestamp, err := time.Parse(time.RFC3339Nano, t.Time)

	if err != nil {
		panic(timestamp)
	}

	point := client.Point{
		Measurement: key,
		Tags: map[string]string{
			"user":      config.MustString("userId"),
			"site":      t.Site,
			"node":      config.Serial(),
			"schema":    t.Schema,
			"channel":   t.Channel,
			"event":     t.Event,
			"thing":     t.Thing,
			"thingType": t.ThingType,
		},
		Fields: map[string]interface{}{},
		Time:   timestamp,
	}

	if t._User != "" {
		point.Tags["user"] = t._User
	}

	for _, p := range t.Points {
		if p.Path == "" {
			point.Fields["value"] = p.Value
		} else {
			point.Fields[p.Path] = p.Value
		}
	}

	bps.Points = append(bps.Points, point)

	//spew.Dump("writing", bps)
	_, err = k.client.Write(bps)
	if err != nil {
		panic(err)
		//return err
	}

	return nil
}
