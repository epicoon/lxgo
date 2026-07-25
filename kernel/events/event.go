// Package events provides the default kernel.IEventManager/kernel.IEvent
// implementations (EventManager/Event) - subscribe/handle named events on
// an EventManager (see kernel.IApp.Events), fire them with Trigger.
package events

import "github.com/epicoon/lxgo/kernel"

/** @interface kernel.IEvent */

// Event is the default kernel.IEvent implementation.
type Event struct {
	app     kernel.IApp
	name    string
	payload kernel.IData
}

var _ kernel.IEvent = (*Event)(nil)

/** @constructor */

// NewEvent constructs an Event with an empty payload.
func NewEvent(app kernel.IApp, name string) *Event {
	return &Event{
		app:     app,
		name:    name,
		payload: kernel.NewEmptyData(),
	}
}

// Name returns the event's name.
func (e *Event) Name() string {
	return e.name
}

// App returns the application the event fired on.
func (e *Event) App() kernel.IApp {
	return e.app
}

// SetPayload sets the event's payload data.
func (e *Event) SetPayload(d kernel.IData) {
	e.payload = d
}

// Payload returns the event's payload data.
func (e *Event) Payload() kernel.IData {
	return e.payload
}
