package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/redigo/redis"
)

var enableSyncFromCloud = config.Bool(false, "homecloud.sync.fromCloud")
var deleteUnknownFromCloud = config.Bool(false, "homecloud.sync.deleteUnknownFromCloud")

var (
	RecordNotFound  = errors.New("Record Not Found")
	RecordUnchanged = errors.New("Record Unchanged")
)

// A wrapper for the connection type.
//
// We need two different connections but the dependency injection
// framework does not provide a way to support this requirement.
//
// So, we use a wrapper type here and inject the wrapper.
type SyncConnection struct {
	Conn *ninja.Connection
}

func newBaseModel(idType string, obj interface{}) baseModel {
	objType := reflect.TypeOf(obj)

	return baseModel{
		syncing:     &sync.WaitGroup{},
		idType:      idType,
		objType:     objType,
		log:         logger.GetLogger(objType.Name() + "Model"),
		entityLocks: make(map[string]*sync.Mutex),
	}
}

type baseModel struct {
	Conn *ninja.Connection `inject:""`

	// A separate connection is required for outbound connections to avoid
	// a dead lock (on the syncing group) that arises if a fetch request
	// (from a client) is received while a sync response (from the cloud)
	// is pending. The inbound request is still blocked, but the blockage
	// will no longer impede the delivery of the inbound response.

	SyncConn *SyncConnection `inject:""`

	syncing     *sync.WaitGroup
	idType      string
	objType     reflect.Type
	log         *logger.Logger
	afterSave   func(obj interface{}, conn redis.Conn) error
	afterDelete func(obj interface{}, conn redis.Conn) error
	onFetch     func(obj interface{}, syncing bool, conn redis.Conn) error
	sendEvent   func(event string, payload interface{}) error

	entityLocks map[string]*sync.Mutex
}

func (m *baseModel) fetch(id string, obj interface{}, syncing bool, conn redis.Conn) error {

	m.log.Debugf("Fetching %s %s", m.idType, id)

	/*	if _, ok := m.entityLocks[id]; !ok {
			m.entityLocks[id] = &sync.Mutex{}
		}

		m.entityLocks[id].Lock()
		m.log.Debugf("Locked %s %s", m.idType, id)
		defer m.entityLocks[id].Unlock()*/

	item, err := redis.Values(conn.Do("HGETALL", m.idType+":"+id))

	if err != nil {
		return err
	}

	if len(item) == 0 {
		return RecordNotFound
	}

	if err := redis.ScanStruct(item, obj); err != nil {
		return err
	}

	if m.onFetch != nil {
		err = m.onFetch(obj, syncing, conn)
	}

	return err
}

func (m *baseModel) Exists(id string, conn redis.Conn) (bool, error) {
	return redis.Bool(conn.Do("EXISTS", m.idType+":"+id))
}

func (m *baseModel) fetchIds(conn redis.Conn) ([]string, error) {

	ids, err := redis.Strings(conn.Do("SMEMBERS", m.idType+"s"))
	m.log.Debugf("Found %d %s id(s)", m.idType, len(ids))

	return ids, err
}

func (m *baseModel) save(id string, obj interface{}, conn redis.Conn) (bool, error) {

	m.log.Debugf("Saving %s %s", m.idType, id)

	defer syncFS()

	existing := reflect.New(m.objType)

	err := m.fetch(id, existing.Interface(), false, conn)

	if err != nil && err != RecordNotFound {
		return false, err
	}

	brandNew := err == RecordNotFound

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

	_, err = conn.Do("HMSET", args...)

	if err != nil {
		return false, fmt.Errorf("Failed to save object %s error:%s", id, err)
	}

	_, err = conn.Do("SADD", m.idType+"s", id)

	if err != nil {
		return false, fmt.Errorf("Failed to add object %s to list of ids error:%s", id, err)
	}

	if m.afterSave != nil {
		err = m.afterSave(obj, conn)

		if err != nil {
			return true, fmt.Errorf("Error during afterSave callback: %s", err)
		}
	}

	if m.sendEvent != nil {
		if brandNew {
			m.sendEvent("created", id)
		} else {
			m.sendEvent("updated", id)
		}
	}

	return true, m.markUpdated(id, time.Now(), conn)
}

func (m *baseModel) delete(id string, conn redis.Conn) error {

	m.log.Debugf("Deleting %s %s", m.idType, id)

	existing := reflect.New(m.objType).Interface()

	existingErr := m.fetch(id, existing, false, conn)
	if existingErr != nil && existingErr != RecordNotFound {
		return fmt.Errorf("Failed fetching existing %s before delete. error:%s", m.idType, existingErr)
	}

	if existingErr == RecordNotFound {

		lastUpdated, _ := m.getLastUpdated(id, conn)
		if lastUpdated == nil {
			// Hasn't ever existed...
			return existingErr
		}
		// At this point we may have a RecordNotFound, but we may as well delete again anyway, just in case
		m.log.Infof("%s id:%s appears to be already deleted, but we'll try again anyway.", m.idType, id)
	}

	defer syncFS()

	conn.Send("MULTI")
	conn.Send("SREM", m.idType+"s", id)
	conn.Send("DEL", m.idType+":"+id)
	_, err := conn.Do("EXEC")

	if m.afterDelete != nil && existingErr == nil {
		err = m.afterDelete(existing, conn)

		if err != nil {
			return fmt.Errorf("Failed on afterDelete: %s", err)
		}
	}

	if m.sendEvent != nil {
		m.sendEvent("deleted", id)
	}

	return m.markUpdated(id, time.Now(), conn)
}

func (m *baseModel) markUpdated(id string, t time.Time, conn redis.Conn) error {
	defer syncFS()

	ts, err := t.MarshalText()
	if err != nil {
		return err
	}
	_, err = conn.Do("HSET", m.idType+"s:updated", id, ts)
	return err
}

func (m *baseModel) getLastUpdated(id string, conn redis.Conn) (*time.Time, error) {
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

// ------- Syncing -------

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

func (m *baseModel) Sync(timeout time.Duration, conn redis.Conn) error {
	m.syncing.Wait()
	m.syncing.Add(1)
	defer m.syncing.Done()

	if config.NoCloud() {
		m.log.Infof("sync: syncing disabled because there is no cloud")
		return nil
	}

	m.log.Infof("sync: Syncing %ss. Save data from cloud?:%s", m.idType, enableSyncFromCloud)

	var diffList SyncDifferenceList

	manifest, err := m.getSyncManifest(conn)
	if err != nil {
		return err
	}

	m.log.Debugf("sync: Sending %d %s local update times", len(*manifest), m.idType)

	calcClient := m.SyncConn.Conn.GetServiceClient("$ninja/services/rpc/modelstore/calculate_sync_items")
	err = calcClient.Call("modelstore.calculate_sync_items", []interface{}{m.idType, manifest}, &diffList, timeout)

	if err != nil {
		return fmt.Errorf("Failed calling calculate_sync_items for model %s error:%s", m.idType, err)
	}

	m.log.Infof("sync: Cloud requires %d %s(s), Node requires %d %s(s)", len(diffList.CloudRequires), m.idType, len(diffList.NodeRequires), m.idType)

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

		err = m.fetch(id, obj, true, conn)

		if err != nil && err != RecordNotFound {
			return fmt.Errorf("Failed retrieving requested %s id:%s error:%s", m.idType, id, err)
		}

		if err == RecordNotFound {
			obj = nil
		}

		lastUpdated, err := m.getLastUpdated(id, conn)
		if err != nil {
			return fmt.Errorf("Failed retrieving last updated time for requested %s id:%s error:%s", m.idType, id, err)
		}

		requestedData[id] = SyncObject{obj, lastUpdated.UnixNano() / int64(time.Millisecond)}
	}

	if deleteUnknownFromCloud {
		for id := range diffList.NodeRequires {
			if _, ok := (*manifest)[id]; !ok { // We've never heard of this, so remove it
				m.log.Infof("Removing %s id:%s from cloud.", m.idType, id)
				requestedData[id] = SyncObject{nil, time.Now().UnixNano() / int64(time.Millisecond)}
			}
		}
	}

	syncClient := m.SyncConn.Conn.GetServiceClient("$ninja/services/rpc/modelstore/do_sync_items")

	var syncReply SyncReply

	err = syncClient.Call("modelstore.do_sync_items", []interface{}{m.idType, requestedData, requestIds}, &syncReply, timeout)

	if err != nil {
		return fmt.Errorf("Failed calling do_sync_items for model %s error:%s", m.idType, err)
	}

	defer syncFS()

	if enableSyncFromCloud {

		for id, requestedObj := range syncReply.RequestedObjects {
			obj := reflect.New(m.objType).Interface()

			err := json.Unmarshal(requestedObj.Data, obj)
			if err != nil {
				m.log.Warningf("Failed to unmarshal requested %s id:%s error: %s", m.idType, id, err)
				m.delete(id, conn)
			} else if string(requestedObj.Data) == "null" {
				m.log.Infof("Requested %s id:%s has been remotely deleted", m.idType, id)
				m.delete(id, conn)
			} else {

				updated, err := m.save(id, obj, conn)
				if err != nil {
					return fmt.Errorf("Failed to save requested %s id:%s error: %s", m.idType, id, err)
				}
				if !updated {
					m.log.Warningf("We requested an updated %s id:%s but it was the same as what we had.", m.idType, id)
				}

			}

			err = m.markUpdated(id, time.Unix(0, requestedObj.LastModified*int64(time.Millisecond)), conn)
			if err != nil {
				m.log.Warningf("Failed to update last modified time of requested %s id:%s error: %s", m.idType, id, err)
			}
		}
	} else {
		m.log.Warningf("Ignoring sync data from cloud.")
	}

	if err != nil {

		ts, err := time.Now().MarshalText()
		if err != nil {
			return err
		}

		_, err = conn.Do("SET", m.idType+"s:synced", ts)

	}

	return err
}

// ClearCloud removes everything from the cloud's version of this model
func (m *baseModel) ClearCloud() error {

	m.log.Infof("Clearing cloud of %ss", m.idType)

	var diffList SyncDifferenceList

	manifest := make(map[string]int64)

	calcClient := m.SyncConn.Conn.GetServiceClient("$ninja/services/rpc/modelstore/calculate_sync_items")
	err := calcClient.Call("modelstore.calculate_sync_items", []interface{}{m.idType, manifest}, &diffList, time.Second*20)

	if err != nil {
		return fmt.Errorf("Failed calling calculate_sync_items for model %s error:%s", m.idType, err)
	}

	requestedData := SyncDataSet{}
	requestIds := make([]string, 0)
	for id := range diffList.NodeRequires {
		requestedData[id] = SyncObject{nil, time.Now().UnixNano() / int64(time.Millisecond)}
	}

	syncClient := m.SyncConn.Conn.GetServiceClient("$ninja/services/rpc/modelstore/do_sync_items")

	var syncReply SyncReply

	err = syncClient.Call("modelstore.do_sync_items", []interface{}{m.idType, requestedData, requestIds}, &syncReply, time.Second*20)

	if err != nil {
		return fmt.Errorf("Failed calling do_sync_items for model %s error:%s", m.idType, err)
	}

	return nil
}

func (m *baseModel) getSyncManifest(conn redis.Conn) (*SyncManifest, error) {

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

func (m *baseModel) SetEventHandler(handler func(event string, payload interface{}) error) {
	m.log.Infof("Got handler! %+v", handler)
	// FIXME: this method should probably be renamed to SetEventSender.
	m.sendEvent = handler
}
