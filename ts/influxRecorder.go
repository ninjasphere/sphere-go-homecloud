package ts

import (
	"fmt"
	"net/url"
	"time"

	"github.com/influxdb/influxdb/client"
	"github.com/ninjasphere/go-ninja/config"
)

type influxRecorder struct {
	client *client.Client // influxdb client
}

func newinfluxRecorder() (*influxRecorder, error) {
	host, err := url.Parse(fmt.Sprintf("http://%s:%d", config.String("localhost", "homecloud.influx.host"), 8086))
	if err != nil {
		return nil, err
	}

	influx, err := client.NewClient(client.Config{URL: *host})

	if err != nil {
		return nil, err
	}

	return &influxRecorder{
		client: influx,
	}, nil

}

func (k *influxRecorder) messageHandler(deliveries <-chan *TimeSeriesPayload) {
	for d := range deliveries {

		start := time.Now()

		err := k.sendTimeseries(d)

		if err != nil {
			log.Errorf("failed to post payload: %s", err)
		}

		log.Debugf("response Time Taken: %v", time.Since(start))

	}
	log.Infof("handle: deliveries channel closed")
}

func (k *influxRecorder) sendTimeseries(t *TimeSeriesPayload) error {

	var key = fmt.Sprintf("%s.%s.%s.%s", t.ThingType, t.Channel, t.Event, t.Site)

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
		Fields:    map[string]interface{}{},
		Time:      time.Unix(0, t.Time*int64(time.Millisecond)),
		Precision: "s",
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

	bps := client.BatchPoints{
		Points:          []client.Point{point},
		Database:        config.String("sphere", "homecloud.influx.database"),
		RetentionPolicy: "default",
	}

	//spew.Dump("writing", len(points), points)
	_, err := k.client.Write(bps)
	if err != nil {
		return err
	}

	//spew.Dump("response", resp, bps)
	return nil
}
