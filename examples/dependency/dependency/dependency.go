package dependency

import (
	"github.com/recallsong/servicehub"
)

// Interface .
type Interface interface {
	Hello(name string) string
}

type config struct {
	Prefix string `file:"prefix"`
}

type provider struct {
	Cfg *config
}

func (p *provider) Hello(name string) string {
	return p.Cfg.Prefix + "hello " + name
}

func init() {
	servicehub.Register("example-dependency-provider", &servicehub.Spec{
		Services:    []string{"example-dependency"},
		Description: "dependency for example",
		ConfigFunc:  func() interface{} { return &config{} },
		Creator: func() servicehub.Provider {
			return &provider{}
		},
	})
}
