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
}

func (p *provider) Init(ctx servicehub.Context) error {
	dep1 := ctx.Service("example-dependency@label").(dependency.Interface)
	fmt.Println(dep1.Hello(p.Cfg.Name))

	dep2 := ctx.Service("example-dependency").(dependency.Interface)
	fmt.Println(dep2.Hello(p.Cfg.Name))
	return nil
}

func init() {
	servicehub.Register("hello-provider", &servicehub.Spec{
		Services:     []string{"hello"},
		Dependencies: []string{"example-dependency@label", "example-dependency"},
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

// OUTPUT:
// INFO[2021-08-25 14:36:45.422] provider example-dependency-provider initialized
// INFO[2021-08-25 14:36:45.422] provider example-dependency-provider@label initialized
// label-hello recallsong
// hello recallsong
// INFO[2021-08-25 14:36:45.422] provider hello-provider (depends services: [example-dependency@label example-dependency]) initialized
// INFO[2021-08-25 14:36:45.423] signals to quit: [hangup interrupt terminated quit]
