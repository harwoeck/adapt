package core

import (
	logger "github.com/harwoeck/liblog/contract"
)

type fieldImpl struct {
	key   string
	value interface{}
}

func (f *fieldImpl) Key() string        { return f.key }
func (f *fieldImpl) Value() interface{} { return f.value }

func field(key string, value interface{}) logger.Field {
	return &fieldImpl{
		key:   key,
		value: value,
	}
}
