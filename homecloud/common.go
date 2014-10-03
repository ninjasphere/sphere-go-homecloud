package homecloud

import (
	"errors"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
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

func (m *baseModel) getSyncManifest() (*SyncManifest, error) {
	conn := m.pool.Get()
	defer conn.Close()

	var manifest SyncManifest = make(map[string]time.Time)

	item, err := redis.Strings(conn.Do("HGETALL", m.idType+":updated"))

	for i := 0; i < len(item); i += 2 {
		t := time.Time{}
		err := t.UnmarshalText([]byte(item[i+1]))
		if err != nil {
			return nil, err
		}
		manifest[item[i]] = t
	}

	if err != nil {
		return nil, err
	}

	/*if err := redis.ScanStruct(item, manifest); err != nil {
		return nil, err
	}*/

	return &manifest, nil
}
