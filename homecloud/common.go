package homecloud

import (
	"errors"

	"github.com/ninjasphere/redigo/redis"
)

var (
	RecordNotFound = errors.New("Record Not Found")
)

type baseModel struct {
	pool   *redis.Pool
	idType string
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

func (m *baseModel) create(id string, obj interface{}) error {
	conn := m.pool.Get()
	defer conn.Close()

	args := redis.Args{}
	args = args.Add(m.idType + ":" + id)
	args = args.AddFlat(obj)

	if _, err := conn.Do("HMSET", args...); err != nil {
		return err
	}

	if _, err := conn.Do("SADD", m.idType+"s", id); err != nil {
		return err
	}

	return nil
}
