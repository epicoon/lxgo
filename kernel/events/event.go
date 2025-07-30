package events

import "github.com/epicoon/lxgo/kernel"

/** @interface kernel.IEvent */
type Event struct {
	app     kernel.IApp
	name    string
	payload kernel.IData
}

/** @constructor */
func NewEvent(app kernel.IApp, name string) *Event {
	return &Event{
		app:     app,
		name:    name,
		payload: kernel.NewEmptyData(),
	}
}

func (e *Event) Name() string {
	return e.name
}

func (e *Event) App() kernel.IApp {
	return e.app
}

func (e *Event) SetPayload(d kernel.IData) {
	e.payload = d
}

func (e *Event) Payload() kernel.IData {
	return e.payload
}
