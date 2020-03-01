package servicehub

import (
	"github.com/recallsong/go-utils/logs"
)

// Context .
type Context interface {
	Hub() *Hub
	Config() interface{}
	Logger() logs.Logger
	Provider(name string, options ...interface{}) interface{}
}

// Creator .
type Creator func() ServiceProvider

// ServiceProviders .
var ServiceProviders = map[string]Creator{}

// RegisterProvider .
func RegisterProvider(name string, creator Creator) {
	ServiceProviders[name] = creator
}

// ServiceProvider .
type ServiceProvider interface {
	Name() string
	Services() []string
	Start() error
	Close() error
}

// ServiceConfigurator .
type ServiceConfigurator interface {
	Config() interface{}
}

// ServiceInitializer .
type ServiceInitializer interface {
	Init(ctx Context) error
}

// ServiceDependencies .
type ServiceDependencies interface {
	Dependencies() []string
}

// ServiceLogger .
type ServiceLogger interface {
	SetLogger(logger logs.Logger)
}

// DependencyProvider .
type DependencyProvider interface {
	Provide(name string, options ...interface{}) interface{}
}
