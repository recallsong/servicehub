package main

import (
	"context"
	"fmt"
	"os"

	"github.com/recallsong/servicehub"
	"github.com/recallsong/servicehub/logs"
)

type config struct {
	Message string `file:"message" flag:"msg" default:"hi" desc:"message to show" env:"HELLO_MESSAGE"`
}

type ExampleService struct {
	Name string
}

type provider struct {
	Cfg     *config
	Log     logs.Logger
	Service *ExampleService `autowired:"test"`
}

func (p *provider) Run(ctx context.Context) error {
	fmt.Println(p.Service)
	return nil
}

func init() {
	servicehub.RegisterGlobalSpec("test-p", &servicehub.Spec{
		Services: []string{"test"},
		Creator: func() servicehub.Provider {
			return &ExampleService{
				Name: "testxxxx",
			}
		},
	})

	servicehub.Register("hello-provider", &servicehub.Spec{
		Services:    []string{"hello"},
		Description: "hello for example",
		ConfigFunc:  func() interface{} { return &config{} },
		Creator: func() servicehub.Provider {
			return &provider{}
		},
	})
}

func main() {
	hub := servicehub.New()
	hub.Run("examples", "", os.Args...)
}
