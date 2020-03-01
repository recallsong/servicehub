package servicehub

import "github.com/recallsong/go-utils/logs"

// Option .
type Option func(hub *Hub)

func processOptions(hub *Hub, opt interface{}) {
	if fn, ok := opt.(Option); ok {
		fn(hub)
	}
}

// WithRequiredServices .
func WithRequiredServices(autoCreate bool, services ...string) interface{} {
	return Option(func(hub *Hub) {
		hub.requires = services
		hub.autoCreate = autoCreate
	})
}

// WithLogger .
func WithLogger(logger logs.Logger) interface{} {
	return Option(func(hub *Hub) {
		hub.logger = logger
	})
}
