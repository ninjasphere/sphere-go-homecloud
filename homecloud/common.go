package homecloud

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/api"
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
}

func (m *baseModel) fetch(id string, obj interface{}) error {
	return m.fetchWithRoot(m.idType, id, obj)
}

func (m *baseModel) fetchWithRoot(idRoot, id string, obj interface{}) error {

	conn := m.pool.Get()
	defer conn.Close()

	item, err := redis.Values(conn.Do("HGETALL", idRoot+":"+id))

	if err != nil {
		return err
	}

	if len(item) == 0 {
		return RecordNotFound
	}

	if err := redis.ScanStruct(item, obj); err != nil {
		return err
	}

	return nil
}

func (m *baseModel) fetchAllIds() ([]string, error) {
	conn := m.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", m.idType+"s"))
}

func (m *baseModel) save(id string, obj interface{}) (updated bool, err error) {
	return m.saveWithRoot(m.idType, id, obj)
}

func (m *baseModel) saveWithRoot(idRoot, id string, obj interface{}) (bool, error) {
	log.Debugf("saveWithRoot", spew.Sdump(idRoot, id, obj))
	conn := m.pool.Get()
	defer conn.Close()

	existing := reflect.New(m.objType)

	err := m.fetchWithRoot(idRoot, id, existing.Interface())
	if err != nil && err != RecordNotFound {
		return false, err
	}

	if err == nil {
		if m.isUnchanged(existing.Interface(), obj) {
			return false, nil
			// XXX: Should this be return RecordUnchanged?
		}
	}

	args := redis.Args{}
	args = args.Add(idRoot + ":" + id)
	args = args.AddFlat(obj)

	if _, err := conn.Do("HMSET", args...); err != nil {
		return false, err
	}

	if _, err := conn.Do("SADD", idRoot+"s", id); err != nil {
		return false, err
	}

	return true, m.markUpdated(idRoot, id)
}

func (m *baseModel) markUpdated(idRoot, id string) error {
	conn := m.pool.Get()
	defer conn.Close()
	t, _ := time.Now().MarshalText()
	_, err := conn.Do("HSET", idRoot+":updated", id, t)
	return err
}

func (m *baseModel) getLastUpdated(idRoot, id string) (int64, error) {
	conn := m.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("HGET", idRoot+":updated", id))
}

func (m *baseModel) isUnchanged(a interface{}, b interface{}) bool {

	spew.Dump("Comparing", a, b)

	aFlat, bFlat := redis.Args{}.AddFlat(a), redis.Args{}.AddFlat(b)

	if len(aFlat) != len(bFlat) {
		log.Infof("Wrong length")
		return false
	}

	for i, x := range aFlat {
		if x != bFlat[i] {
			log.Infof("val %s != %s", x, bFlat[i])
			return false
		}
	}

	log.Infof("Equal!")
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

type SyncReply struct {
	Model            string      `json:"model"`
	RequestedObjects SyncDataSet `json:"requestedObjects"`
	PushedObjects    SyncDataSet `json:"pushedObjects"`
}

func (m *baseModel) sync() error {

	var diffList SyncDifferenceList

	manifest, err := m.getSyncManifest()
	if err != nil {
		return err
	}

	calcClient := m.conn.GetServiceClient("$ninja/services/rpc/modelstore/calculate_sync_items")
	err = calcClient.Call("modelstore.calculate_sync_items", []interface{}{m.idType, manifest}, &diffList, time.Second*20)

	spew.Dump("modelstore.calculate_sync_items REPLY", err, diffList)

	if err != nil {
		return fmt.Errorf("Failed calling calculate_sync_items for model %s error:%s", m.idType, err)
	}

	var requestIds []string
	for id := range diffList.NodeRequires {
		requestIds = append(requestIds, id)
	}

	requestedData := SyncDataSet{}

	for id := range diffList.CloudRequires {
		obj := reflect.New(m.objType)
		err = m.fetch(id, obj)
		if err != nil {
			return fmt.Errorf("Failed retrieving requested %s id:%s error:%s", m.idType, id, err)
		}

		lastUpdated, err := m.getLastUpdated(m.idType, id)
		if err != nil {
			return fmt.Errorf("Failed retrieving last updated time for requested %s id:%s error:%s", m.idType, id, err)
		}

		requestedData[id] = SyncObject{obj, lastUpdated}
	}

	syncClient := m.conn.GetServiceClient("$ninja/services/rpc/modelstore/do_sync_items")

	var syncReply SyncReply

	err = syncClient.Call("modelstore.do_sync_items", []interface{}{m.idType, requestedData, requestIds}, &syncReply, time.Second*20)

	spew.Dump("modelstore.do_sync_items REPLY", err, syncReply)

	if err != nil {
		return fmt.Errorf("Failed calling do_sync_items for model %s error:%s", m.idType, err)
	}

	return err
}

func (m *baseModel) getSyncManifest() (*SyncManifest, error) {
	conn := m.pool.Get()
	defer conn.Close()

	var manifest SyncManifest = make(map[string]int64)

	item, err := redis.Strings(conn.Do("HGETALL", m.idType+":updated"))

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

	/*if err := redis.ScanStruct(item, manifest); err != nil {
		return nil, err
	}*/

	return &manifest, nil
}
