package main

import (
	"os"
	"time"

	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/servicehub"
)

type subConfig struct {
	Name string `file:"name" flag:"hello_name" default:"recallsong" desc:"name to show"`
}

type config struct {
	Message   string    `file:"message" flag:"msg" default:"hi" desc:"message to show"`
	SubConfig subConfig `file:"sub"`
}

type define struct{}

func (d *define) Service() []string      { return []string{"hello"} }
func (d *define) Dependencies() []string { return []string{} }
func (d *define) Description() string    { return "hello for example" }
func (d *define) Config() interface{}    { return &config{} }
func (d *define) Creator() servicehub.Creator {
	return func() servicehub.Provider {
		return &provider{}
	}
}

type provider struct {
	C       *config
	L       logs.Logger
	closeCh chan struct{}
}

func (p *provider) Init(ctx servicehub.Context) error {
	p.L.Info("message: ", p.C.Message)
	p.closeCh = make(chan struct{})
	return nil
}

func (p *provider) Start() error {
	p.L.Info("now hello provider is running...")
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			p.L.Info("do something...")
		case <-p.closeCh:
			return nil
		}
	}
}

func (p *provider) Close() error {
	p.L.Info("now hello provider is closing...")
	close(p.closeCh)
	return nil
}

func init() {
	servicehub.RegisterProvider("hello", &define{})
}

func main() {
	hub := servicehub.New()
	hub.Run("examples", os.Args...)
}
