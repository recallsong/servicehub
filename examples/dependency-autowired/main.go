package main

import (
	"fmt"
	"os"

	"github.com/recallsong/servicehub"
	"github.com/recallsong/servicehub/examples/dependency/dependency"
	"github.com/recallsong/servicehub/logs"
)

type config struct {
	Name string `file:"name" default:"recallsong"`
}

type provider struct {
	Cfg *config
	Log logs.Logger
	Dep dependency.Interface `autowired:"example-dependency"`
}

func (p *provider) Init(ctx servicehub.Context) error {
	fmt.Println(p.Dep.Hello(p.Cfg.Name))
	return nil
}

func init() {
	servicehub.Register("hello-provider", &servicehub.Spec{
		Services:     []string{"hello"},
		Dependencies: []string{"example-dependency"},
		Description:  "hello for example",
		ConfigFunc:   func() interface{} { return &config{} },
		Creator: func() servicehub.Provider {
			return &provider{}
		},
	})
}

func main() {
	hub := servicehub.New()
	hub.Run("examples", "", os.Args...)
}
