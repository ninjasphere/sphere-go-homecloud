package homecloud

import (
	"errors"

	"github.com/ninjasphere/redigo/redis"
)

var (
	RecordNotFound = errors.New("Record Not Found")
)

type baseModel struct {
	conn   redis.Conn
	idType string
}

func (m *baseModel) fetch(id string, obj interface{}) error {

	item, err := redis.Values(m.conn.Do("HGETALL", m.idType+":"+id))

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
	return redis.Strings(m.conn.Do("SMEMBERS", m.idType+"s"))
}

func (m *baseModel) create(id string, obj interface{}) error {

	args := redis.Args{}
	args = args.Add(m.idType + ":" + id)
	args = args.AddFlat(obj)

	if _, err := m.conn.Do("HMSET", args...); err != nil {
		return err
	}

	if _, err := m.conn.Do("SADD", m.idType+"s", id); err != nil {
		return err
	}

	return nil
}
