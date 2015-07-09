package ts

import (
	"fmt"
	"math/rand"
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

	x := time.Now().AddDate(-2, 0, 0)

	for i := 0; i < 10000; i++ {
		var key string

		points := []client.Point{}

		for _, v := range t.Points {

			if v.Path == "" {
				// sphere.timeseries.161139cf-acd8-11e4-98cf-883314fa95ac.161139cf-acd8-11e4-98cf-883314fa95ac.on-off.state
				key = fmt.Sprintf("sphere.%s.%s.%s.%s", t.Site, t.Thing, t.Channel, t.Event)
			} else {
				// sphere.timeseries.161139cf-acd8-11e4-98cf-883314fa95ac.161139cf-acd8-11e4-98cf-883314fa95ac.light.state.hue
				key = fmt.Sprintf("sphere.%s.%s.%s.%s.%s", t.Site, t.Thing, t.Channel, t.Event, v.Path)
			}

			for i := 0; i < 20; i++ {

				x = x.Add(time.Minute)

				point := client.Point{
					Measurement: key,
					Tags: map[string]string{
						"type":      v.Type,
						"schema":    t.Schema,
						"thingType": t.ThingType,
					},
					Fields: map[string]interface{}{
						"value": v.Value.(float64) + (rand.Float64() * 20) - 10,
					},
					Time:      x.Add(time.Second),
					Precision: "s",
				}

				points = append(points, point)

				//spew.Dump(point)
			}

		}

		bps := client.BatchPoints{
			Points:          points,
			Database:        config.String("sphere", "homecloud.influx.database"),
			RetentionPolicy: "default",
		}

		//spew.Dump("writing", len(points), points)
		_, err := k.client.Write(bps)
		if err != nil {
			return err
		}
	}
	//spew.Dump("response", resp, bps)
	return nil
}
