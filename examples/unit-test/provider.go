package example

import (
	"github.com/recallsong/servicehub"
)

// Interface .
type Interface interface {
	Hello(name string) string
	Add(a, b int) int
}

var _ Interface = (*provider)(nil) // check interface implemented

type provider struct{}

func (p *provider) Hello(name string) string {
	return "hello " + name
}

func (p *provider) Add(a, b int) int {
	return a + b
}

func (p *provider) sub(a, b int) int {
	return a - b
}

func init() {
	servicehub.Register("example-provider", &servicehub.Spec{
		Services:    []string{"example"},
		Description: "example",
		Creator: func() servicehub.Provider {
			return &provider{}
		},
	})
}
