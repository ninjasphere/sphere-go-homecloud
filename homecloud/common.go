package homecloud

import (
	"errors"
	"reflect"
	"time"

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

	conn := m.pool.Get()
	defer conn.Close()

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

	return nil
}

func (m *baseModel) fetchAllIds() ([]string, error) {
	conn := m.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", m.idType+"s"))
}

func (m *baseModel) save(id string, obj interface{}) error {
	return m.saveWithRoot(m.idType, id, obj)
}

func (m *baseModel) saveWithRoot(idRoot, id string, obj interface{}) error {
	conn := m.pool.Get()
	defer conn.Close()

	existing := reflect.New(m.objType)

	err := m.fetch(id, existing.Interface())
	if err != nil && err != RecordNotFound {
		return err
	}

	if err == nil {
		if m.isUnchanged(existing.Interface(), obj) {
			return nil
			// XXX: Should this be return RecordUnchanged?
		}
	}

	args := redis.Args{}
	args = args.Add(idRoot + ":" + id)
	args = args.AddFlat(obj)

	if _, err := conn.Do("HMSET", args...); err != nil {
		return err
	}

	if _, err := conn.Do("SADD", idRoot+"s", id); err != nil {
		return err
	}

	if _, err := conn.Do("HSET", "updated", idRoot+":"+id, time.Now()); err != nil {
		return err
	}

	return nil
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

func (m *baseModel) update(id string, obj interface{}) error {
	conn := m.pool.Get()
	defer conn.Close()

	args := redis.Args{}
	args = args.Add(m.idType + ":" + id)
	args = args.AddFlat(obj)

	if _, err := conn.Do("HMSET", args...); err != nil {
		return err
	}

	return nil
}
