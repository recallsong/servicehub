package servicehub

import (
	"fmt"
	"os"

	"github.com/recallsong/go-utils/logs"
)

// Creator .
type Creator func() Provider

// ProviderDefine .
type ProviderDefine interface {
	Service() []string
	Creator() Creator
}

// ProviderUsage .
type ProviderUsage interface {
	Summary() string
	Description() string
}

// ServiceDependencies .
type ServiceDependencies interface {
	Dependencies() []string
}

// ConfigCreator .
type ConfigCreator interface {
	Config() interface{}
}

// serviceProviders .
var serviceProviders = map[string]ProviderDefine{}

// RegisterProvider .
func RegisterProvider(name string, define ProviderDefine) {
	if _, ok := serviceProviders[name]; ok {
		fmt.Printf("provider %s already exist\n", name)
		os.Exit(-1)
	}
	serviceProviders[name] = define
}

// Provider .
type Provider interface {
	Start() error
	Close() error
}

// Context .
type Context interface {
	Hub() *Hub
	Config() interface{}
	Logger() logs.Logger
	Service(name string, options ...interface{}) interface{}
}

// ProviderInitializer .
type ProviderInitializer interface {
	Init(ctx Context) error
}

// DependencyProvider .
type DependencyProvider interface {
	Provide(name string, options ...interface{}) interface{}
}
