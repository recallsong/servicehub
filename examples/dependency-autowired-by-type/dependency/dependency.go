package dependency

import (
	"reflect"

	"github.com/recallsong/servicehub"
)

// Interface .
type Interface interface {
	Hello(name string) string
}

type provider struct{}

func (p *provider) Hello(name string) string {
	return "hello " + name
}

func init() {
	servicehub.Register("example-dependency-provider", &servicehub.Spec{
		Services:    []string{"example-dependency"},
		Types:       []reflect.Type{reflect.TypeOf((*Interface)(nil)).Elem()},
		Description: "dependency for example",
		Creator: func() servicehub.Provider {
			return &provider{}
		},
	})
}
