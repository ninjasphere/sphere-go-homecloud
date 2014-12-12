package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
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
	Conn       *ninja.Connection `inject:""`
	lastStates map[string]*LastState
}

func NewStateManager() StateManager {

	return &NinjaStateManager{
		lastStates: make(map[string]*LastState),
		log:        logger.GetLogger("sphere-go-homecloud-state"),
	}
}

func (sm *NinjaStateManager) PostConstruct() error {
	go sm.startListener()
	return nil
}

func (sm *NinjaStateManager) Merge(thing *model.Thing) {

	deviceID := thing.DeviceID

	// no point going futher, we have an empty model
	if deviceID == nil {
		sm.log.Infof(spew.Sprintf("bad thing %v", thing))
		return
	}

	if thing.Device != nil && thing.Device.Channels != nil {
		for _, channelModel := range *thing.Device.Channels {

			key := fmt.Sprintf("%s-%s", *deviceID, channelModel.ID)

			sm.log.Debugf("channel key %s state %v", key, sm.lastStates[key])

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

	err := sm.Conn.GetServiceClient("$device/:deviceid/channel/:channelid").OnEvent("state", func(params *json.RawMessage, values map[string]string) bool {

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
