package servicehub

// Events events about Hub
type Events interface {
	Initialized() <-chan error
	Started() <-chan error
	Exited() <-chan error
}

type events struct {
	_initialized bool
	_started     bool
	initialized  chan error
	started      chan error
	exited       chan error
}

func newEvents() *events {
	return &events{
		initialized: make(chan error, 1),
		started:     make(chan error, 1),
		exited:      make(chan error, 1),
	}
}

func (e *events) Initialized() <-chan error {
	return e.initialized
}

func (e *events) Started() <-chan error {
	return e.started
}

func (e *events) Exited() <-chan error {
	return e.exited
}

// Events return Events
func (h *Hub) Events() Events {
	events := newEvents()
	h.listeners = append(h.listeners, &DefaultListener{
		AfterInitFunc: func(h *Hub) error {
			events._initialized = true
			close(events.initialized)
			return nil
		},
		AfterStartFunc: func(h *Hub) error {
			events._started = true
			close(events.started)
			return nil
		},
		BeforeExitFunc: func(h *Hub, err error) error {
			if !events._initialized {
				events.initialized <- err
				close(events.initialized)
			}
			if !events._started {
				events.started <- err
				close(events.started)
			}
			events.exited <- err
			close(events.exited)
			return nil
		},
	})
	return events
}
