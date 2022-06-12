package main

import (
	"context"
	"fmt"
	"time"

	"github.com/recallsong/servicehub"
	"github.com/recallsong/servicehub/logs"
)

type config struct {
	Error bool `file:"error"`
}

type provider struct {
	Cfg *config
	Log logs.Logger
}

func (p *provider) Run(ctx context.Context) error {
	if p.Cfg.Error {
		time.Sleep(3 * time.Second)
		return fmt.Errorf("run error")
	}
	p.Log.Info("run with no error")
	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
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
	servicehub.Run(&servicehub.RunOptions{
		Content: `
hello-provider:
    error: false
hello-provider@error1:
    error: true
hello-provider@error2:
    error: true
`,
	})
}
