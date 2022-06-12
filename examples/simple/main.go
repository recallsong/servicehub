package main

import (
	"os"

	"github.com/recallsong/servicehub"
	"github.com/recallsong/servicehub/logs"
)

type config struct {
	Message   string    `file:"message" flag:"msg" default:"hi" desc:"message to show" env:"HELLO_MESSAGE"`
	SubConfig subConfig `file:"sub"`
}

type subConfig struct {
	Name string `file:"name" flag:"hello_name" default:"recallsong" desc:"name to show"`
}

type provider struct {
	Cfg *config     // auto inject this field
	Log logs.Logger // auto inject this field
}

func init() {
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
