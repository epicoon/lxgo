package events

import "github.com/epicoon/lxgo/kernel"

/** @interface kernel.IEventManager */

// EventManager is the default kernel.IEventManager implementation.
type EventManager struct {
	app           kernel.IApp
	events        map[string][]kernel.FEventHandler
	eventHandlers map[string][]kernel.IEventHandler
}

var _ kernel.IEventManager = (*EventManager)(nil)

/** @constructor */

// NewEventManager constructs an EventManager bound to app.
func NewEventManager(app kernel.IApp) *EventManager {
	return &EventManager{app: app}
}

// Subscribe registers a function to run when eventName fires.
func (em *EventManager) Subscribe(eventName string, handler kernel.FEventHandler) {
	if em.events == nil {
		em.events = make(map[string][]kernel.FEventHandler)
	}
	if em.events[eventName] == nil {
		em.events[eventName] = make([]kernel.FEventHandler, 0, 1)
	}
	em.events[eventName] = append(em.events[eventName], handler)
}

// Handle registers an IEventHandler to run when eventName fires, binding it to the manager's app.
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

// Trigger fires eventName, running every subscribed function and IEventHandler with the given payload data.
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
