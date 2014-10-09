package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
)

// struct date, payload
type LastState struct {
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// merge state

type StateManager interface {
	Merge(thing *model.Thing)
	Reset()
}

type NinjaStateManager struct {
	sync.Mutex
	log        *logger.Logger
	conn       *ninja.Connection
	lastStates map[string]*LastState
}

func NewStateManager(conn *ninja.Connection) StateManager {

	sm := &NinjaStateManager{
		lastStates: make(map[string]*LastState),
		conn:       conn,
		log:        logger.GetLogger("sphere-go-homecloud-state"),
	}

	go sm.startListener()

	return sm
}

func (sm *NinjaStateManager) Merge(thing *model.Thing) {

	deviceID := thing.DeviceID

	if thing.Device != nil && thing.Device.Channels != nil {
		for _, channelModel := range *thing.Device.Channels {

			key := fmt.Sprintf("%s-%s", *deviceID, channelModel.ID)

			sm.log.Infof("channel key %s state %v", key, sm.lastStates[key])

			if val, ok := sm.lastStates[key]; ok {
				channelModel.LastState = val
			}
		}
	}

}

func (sm *NinjaStateManager) Reset() {
	defer sm.Unlock()
	sm.Lock()
	sm.lastStates = make(map[string]*LastState)
}

func (sm *NinjaStateManager) startListener() {

	sm.log.Infof("startListener")

	err := sm.conn.GetServiceClient("$device/:deviceid/channel/:channelid").OnEvent("state", func(params *json.RawMessage, values map[string]string) bool {

		var data interface{}

		err := json.Unmarshal(*params, &data)

		if err != nil {
			sm.log.Errorf("bad content: %s", err)
		}

		key := fmt.Sprintf("%s-%s", values["deviceid"], values["channelid"])

		sm.lastStates[key] = &LastState{
			Timestamp: int64(time.Now().UnixNano() / 1e6),
			Payload:   data,
		}

		return true
	})

	if err != nil {
		sm.log.FatalError(err, "cant register service")
	}

}
