package events

import "github.com/epicoon/lxgo/kernel"

/** @interface kernel.IEventManager */
type EventManager struct {
	app           kernel.IApp
	events        map[string][]kernel.FEventHandler
	eventHandlers map[string][]kernel.IEventHandler
}

/** @constructor */
func NewEventManager(app kernel.IApp) *EventManager {
	return &EventManager{app: app}
}

func (em *EventManager) Subscribe(eventName string, handler kernel.FEventHandler) {
	if em.events == nil {
		em.events = make(map[string][]kernel.FEventHandler)
	}
	if em.events[eventName] == nil {
		em.events[eventName] = make([]kernel.FEventHandler, 0, 1)
	}
	em.events[eventName] = append(em.events[eventName], handler)
}

func (em *EventManager) Handle(eventName string, handler kernel.IEventHandler) {
	if em.eventHandlers == nil {
		em.eventHandlers = make(map[string][]kernel.IEventHandler)
	}
	if em.eventHandlers[eventName] == nil {
		em.eventHandlers[eventName] = make([]kernel.IEventHandler, 0, 1)
	}
	handler.SetApp(em.app)
	em.eventHandlers[eventName] = append(em.eventHandlers[eventName], handler)
}

func (em *EventManager) Trigger(eventName string, d ...kernel.IData) {
	e := NewEvent(em.app, eventName)
	if len(d) == 1 {
		e.SetPayload(d[0])
	}

	funcs, ok := em.events[eventName]
	if ok {
		for _, f := range funcs {
			f(e)
		}
	}
	handlers, ok := em.eventHandlers[eventName]
	if ok {
		for _, handler := range handlers {
			handler.Run(e)
		}
	}
}
