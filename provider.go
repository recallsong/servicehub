package servicehub

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/recallsong/servicehub/logs"
)

// Creator .
type Creator func() Provider

// ProviderDefine .
type ProviderDefine interface {
	Creator() Creator
}

// ProviderServices .
type ProviderServices interface {
	Services() []string
}

// ServiceTypes .
type ServiceTypes interface {
	Types() []reflect.Type
}

// ProviderUsageSummary .
type ProviderUsageSummary interface {
	Summary() string
}

// ProviderUsage .
type ProviderUsage interface {
	Description() string
}

// ServiceDependencies .
type ServiceDependencies interface {
	Dependencies(*Hub) []string
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
type Provider interface{}

// Context .
type Context interface {
	context.Context
	Hub() *Hub
	Config() interface{}
	Logger() logs.Logger
	Service(name string, options ...interface{}) interface{}
	AddTask(task func(context.Context) error, options ...TaskOption)
	Key() string
	Label() string
	Provider() Provider
}

// TaskOption .
type TaskOption func(*task)

// ProviderRunner .
type ProviderRunner interface {
	Start() error
	Close() error
}

// ProviderRunnerWithContext .
type ProviderRunnerWithContext interface {
	Run(context.Context) error
}

// ProviderInitializer .
type ProviderInitializer interface {
	Init(ctx Context) error
}

// DependencyContext .
type DependencyContext interface {
	Type() reflect.Type
	Tags() reflect.StructTag
	Service() string
	Key() string
	Label() string
	Caller() string
	CallerLabel() string
}

// DependencyProvider .
type DependencyProvider interface {
	Provide(ctx DependencyContext, options ...interface{}) interface{}
}
