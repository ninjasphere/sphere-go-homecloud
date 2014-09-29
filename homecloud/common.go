package homecloud

import "github.com/ninjasphere/redigo/redis"

type baseModel struct {
	conn   redis.Conn
	idType string
}

func (m *baseModel) fetch(id string, obj interface{}) error {

	item, err := redis.Values(m.conn.Do("HGETALL", m.idType+":"+id))

	if err != nil {
		return err
	}

	if err := redis.ScanStruct(item, obj); err != nil {
		return err
	}

	return nil
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
