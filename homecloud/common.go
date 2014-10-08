package homecloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/redigo/redis"
)

var (
	RecordNotFound  = errors.New("Record Not Found")
	RecordUnchanged = errors.New("Record Unchanged")
)

type baseModel struct {
	pool    *redis.Pool
	idType  string
	objType reflect.Type
	conn    *ninja.Connection
	log     *logger.Logger
}

func (m *baseModel) fetch(id string, obj interface{}) error {

	m.log.Debugf("Fetching %s %s", m.idType, id)

	conn := m.pool.Get()
	defer conn.Close()

	item, err := redis.Values(conn.Do("HGETALL", m.idType+":"+id))

	if err != nil {
		return err
	}

	if len(item) == 0 {
		m.log.Debugf("%s %s was not found", m.idType, id)
		return RecordNotFound
	}

	if err := redis.ScanStruct(item, obj); err != nil {
		return err
	}

	return nil
}

func (m *baseModel) fetchIds() ([]string, error) {

	conn := m.pool.Get()
	defer conn.Close()
	ids, err := redis.Strings(conn.Do("SMEMBERS", m.idType+"s"))
	m.log.Debugf("Found %d %s id(s)", m.idType, len(ids))

	return ids, err
}

func (m *baseModel) save(id string, obj interface{}) (bool, error) {

	m.log.Debugf("Saving %s %s", m.idType, id)

	conn := m.pool.Get()
	defer conn.Close()

	existing := reflect.New(m.objType)

	err := m.fetch(id, existing.Interface())
	if err != nil && err != RecordNotFound {
		return false, err
	}

	if err == nil {
		if m.isUnchanged(existing.Interface(), obj) {
			m.log.Debugf("%s %s was unchanged.", m.idType, id)

			return false, nil
			// XXX: Should this be return RecordUnchanged?
		}
	}

	args := redis.Args{}
	args = args.Add(m.idType + ":" + id)
	args = args.AddFlat(obj)

	if _, err := conn.Do("HMSET", args...); err != nil {
		return false, err
	}

	if _, err := conn.Do("SADD", m.idType+"s", id); err != nil {
		return false, err
	}

	return true, m.markUpdated(id, time.Now())
}

func (m *baseModel) delete(id string) error {

	m.log.Debugf("Deleting %s %s", m.idType, id)

	conn := m.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("SREM", m.idType+"s", id)
	conn.Send("DEL", m.idType+":"+id)
	_, err := conn.Do("EXEC")

	return err

	// TODO: announce deletion via MQTT
	// publish(Ninja.topics.room.goodbye.room(roomId)
	// publish(Ninja.topics.location.calibration.delete, {zone: roomId})

}

func (m *baseModel) markUpdated(id string, t time.Time) error {
	conn := m.pool.Get()
	defer conn.Close()
	ts, err := t.MarshalText()
	if err != nil {
		return err
	}
	_, err = conn.Do("HSET", m.idType+"s:updated", id, ts)
	return err
}

func (m *baseModel) getLastUpdated(id string) (*time.Time, error) {
	conn := m.pool.Get()
	defer conn.Close()
	timeString, err := redis.String(conn.Do("HGET", m.idType+"s:updated", id))

	if err != nil || timeString == "" {
		return nil, err
	}

	t := time.Time{}
	err = t.UnmarshalText([]byte(timeString))
	return &t, err
}

func (m *baseModel) isUnchanged(a interface{}, b interface{}) bool {

	aFlat, bFlat := redis.Args{}.AddFlat(a), redis.Args{}.AddFlat(b)

	if len(aFlat) != len(bFlat) {
		return false
	}

	for i, x := range aFlat {
		if x != bFlat[i] {
			return false
		}
	}

	return true
}

/*
The manifest needs to be sent via rpc
conn.GetServiceClient("$ninja/services/rpc/modelstore").call("calculate_sync_items", manifest, &whateverTheReplyIs, time.Second * 20) )
*/

// mosquitto_pub -m '{"id":123, "params": ["device",{}],"jsonrpc": "2.0","method":"modelstore.calculate_sync_items","time":132123123}' -t '$ninja/services/rpc/modelstore/calculate_sync_items'

type SyncDifferenceList struct {
	Model         string       `json:"model"`
	CloudRequires SyncManifest `json:"cloud_requires"`
	NodeRequires  SyncManifest `json:"node_requires"`
}

type SyncManifest map[string]int64

type SyncDataSet map[string]interface{}

type SyncObject struct {
	Data         interface{} `json:"data"`
	LastModified int64       `json:"last_modified"`
}

type SyncRawObject struct {
	Data         json.RawMessage `json:"data"`
	LastModified int64           `json:"last_modified"`
}

type SyncReply struct {
	Model            string                   `json:"model"`
	RequestedObjects map[string]SyncRawObject `json:"requestedObjects"`
	PushedObjects    SyncDataSet              `json:"pushedObjects"`
}

func (m *baseModel) sync() error {

	log.Infof("sync: Syncing %ss", m.idType)

	var diffList SyncDifferenceList

	manifest, err := m.getSyncManifest()
	if err != nil {
		return err
	}

	log.Debugf("sync: Sending %d %s local update times", len(*manifest), m.idType)

	calcClient := m.conn.GetServiceClient("$ninja/services/rpc/modelstore/calculate_sync_items")
	err = calcClient.Call("modelstore.calculate_sync_items", []interface{}{m.idType, manifest}, &diffList, time.Second*20)

	if err != nil {
		return fmt.Errorf("Failed calling calculate_sync_items for model %s error:%s", m.idType, err)
	}

	log.Infof("sync: Cloud requires %d %s(s), Node requires %d %s(s)", len(diffList.CloudRequires), m.idType, len(diffList.NodeRequires), m.idType)

	if len(diffList.CloudRequires)+len(diffList.NodeRequires) == 0 {
		// Nothing to do, we're in sync.
		return nil
	}

	requestIds := make([]string, 0)
	for id := range diffList.NodeRequires {
		requestIds = append(requestIds, id)
	}

	requestedData := SyncDataSet{}

	for id := range diffList.CloudRequires {
		obj := reflect.New(m.objType).Interface()
		err = m.fetch(id, obj)

		if err != nil {
			return fmt.Errorf("Failed retrieving requested %s id:%s error:%s", m.idType, id, err)
		}

		lastUpdated, err := m.getLastUpdated(id)
		if err != nil {
			return fmt.Errorf("Failed retrieving last updated time for requested %s id:%s error:%s", m.idType, id, err)
		}

		requestedData[id] = SyncObject{obj, lastUpdated.UnixNano() / int64(time.Millisecond)}
	}

	syncClient := m.conn.GetServiceClient("$ninja/services/rpc/modelstore/do_sync_items")

	var syncReply SyncReply

	err = syncClient.Call("modelstore.do_sync_items", []interface{}{m.idType, requestedData, requestIds}, &syncReply, time.Second*20)

	if err != nil {
		return fmt.Errorf("Failed calling do_sync_items for model %s error:%s", m.idType, err)
	}

	for id, requestedObj := range syncReply.RequestedObjects {
		obj := reflect.New(m.objType).Interface()

		err := json.Unmarshal(requestedObj.Data, obj)
		if err != nil {
			log.Warningf("Failed to unmarshal requested %s id:%s error: %s", m.idType, id, err)
			m.delete(id)
		} else if string(requestedObj.Data) == "null" {
			log.Infof("Requested %s id:%s has been remotely deleted", m.idType, id)
			m.delete(id)
		} else {

			updated, err := m.save(id, obj)
			if err != nil {
				return fmt.Errorf("Failed to save requested %s id:%s error: %s", m.idType, id, err)
			}
			if !updated {
				log.Warningf("We requested an updated %s id:%s but it was the same as what we had.", m.idType, id)
			}

		}

		err = m.markUpdated(id, time.Unix(0, requestedObj.LastModified*int64(time.Millisecond)))
		if err != nil {
			log.Warningf("Failed to update last modified time of requested %s id:%s error: %s", m.idType, id, err)
		}
	}

	return err
}

// ClearCloud removes everything from the cloud's version of this model
func (m *baseModel) ClearCloud() error {

	log.Infof("Clearing cloud of %ss", m.idType)

	var diffList SyncDifferenceList

	manifest := make(map[string]int64)

	calcClient := m.conn.GetServiceClient("$ninja/services/rpc/modelstore/calculate_sync_items")
	err := calcClient.Call("modelstore.calculate_sync_items", []interface{}{m.idType, manifest}, &diffList, time.Second*20)

	if err != nil {
		return fmt.Errorf("Failed calling calculate_sync_items for model %s error:%s", m.idType, err)
	}

	requestedData := SyncDataSet{}
	requestIds := make([]string, 0)
	for id := range diffList.NodeRequires {
		requestedData[id] = SyncObject{nil, time.Now().UnixNano() / int64(time.Millisecond)}
	}

	syncClient := m.conn.GetServiceClient("$ninja/services/rpc/modelstore/do_sync_items")

	var syncReply SyncReply

	err = syncClient.Call("modelstore.do_sync_items", []interface{}{m.idType, requestedData, requestIds}, &syncReply, time.Second*20)

	if err != nil {
		return fmt.Errorf("Failed calling do_sync_items for model %s error:%s", m.idType, err)
	}

	return nil
}

func (m *baseModel) getSyncManifest() (*SyncManifest, error) {
	conn := m.pool.Get()
	defer conn.Close()

	var manifest SyncManifest = make(map[string]int64)

	item, err := redis.Strings(conn.Do("HGETALL", m.idType+"s:updated"))

	for i := 0; i < len(item); i += 2 {
		t := time.Time{}
		err := t.UnmarshalText([]byte(item[i+1]))
		if err != nil {
			return nil, err
		}
		manifest[item[i]] = t.UnixNano() / int64(time.Millisecond)
	}

	if err != nil {
		return nil, err
	}

	return &manifest, nil
}
