package event

import (
	"game-data-summary/entity"
	"reflect"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

type Event struct {
	T reflect.Type
	C func(data interface{})
}

type EventMgr struct {
	//key:event name
	Events map[string]*Event
}

// 事件触发
func (em *EventMgr) OnEvent(msg *entity.Msg) {
	event := em.Events[msg.Event]
	if event != nil {
		data := reflect.New(event.T).Interface()
		if err := jsoniter.UnmarshalFromString(msg.Data, &data); err != nil {
			zap.L().Error("消息解析失败", zap.Any("err", err))
			return
		}
		go func() {
			defer func() {
				if err := recover(); err != nil {
					zap.L().Error("panic", zap.Any("err", err))
				}
			}()
			event.C(data)
		}()
	} else {
		zap.L().Error("无法找到事件处理对象", zap.Any("e", msg))
	}
}

// 事件注册
func (em *EventMgr) Register(e string, t reflect.Type, f func(data interface{})) {
	em.Events[e] = &Event{
		T: t,
		C: f,
	}
}
