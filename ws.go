package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/gorilla/websocket"
	"github.com/ninjasphere/go-ninja/api"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NewWebsocketServer(conn *ninja.Connection) {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		topic, err := url.QueryUnescape(r.URL.Path[1:])

		if err != nil {
			log.Errorf("Websocket invalid topic: %s : %s", r.URL.Path, err)
			return
		}

		log.Debugf("Websocket incoming URL: %s", topic)

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Errorf("Websocket upgrade error: %s", err)
			return
		}

		conn.SubscribeRaw(topic, func(payload *json.RawMessage, headers map[string]string) bool {

			var outgoing = []interface{}{payload, headers}
			json, err := json.Marshal(outgoing)

			if err != nil {
				log.Errorf("Websocket failed to marshal outgoing message: %s", err)
			} else if err = ws.WriteMessage(websocket.TextMessage, json); err != nil {
				log.Errorf("Websocket failed writing message error: %s", err)
			}

			return true
		})

		go func() {
			for {
				_, p, err := ws.ReadMessage()
				if err != nil {
					return
				}

				log.Debugf("Incoming ws message: %s to %s", p, topic)

				conn.GetMqttClient().Publish(mqtt.QoS(0), topic, p)

				if err != nil {
					log.Errorf("Websocket failed publishing message to mqtt: %s", err)
				}

			}
		}()

	})

	err := http.ListenAndServe(":9001", nil)
	if err != nil {
		log.Errorf("Failed to start websocket server: %s", err)
	}

}
