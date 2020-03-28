package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/servicehub"
)

type config struct {
	Addr string `file:"addr" flag:"pprof_addr" env:"PPROF_ADDR" default:":6580" desc:"server address to listen"`
}

type providerDefine struct{}

func (d *providerDefine) Service() []string {
	return []string{"pprof"}
}

func (d *providerDefine) Summary() string {
	return "start pprof http server"
}

func (d *providerDefine) Description() string {
	return d.Summary()
}

func (d *providerDefine) Creator() servicehub.Creator {
	return newProvider
}

func (d *providerDefine) Config() interface{} {
	return &config{}
}

// provider .
type provider struct {
	Cfg    *config
	Logger logs.Logger
	server *http.Server
}

// New .
func newProvider() servicehub.Provider {
	return &provider{}
}

// Init .
func (p *provider) Init(ctx servicehub.Context) error {
	server := &http.Server{Addr: p.Cfg.Addr}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	server.Handler = mux
	p.server = server
	return nil
}

// Start .
func (p *provider) Start() error {
	p.Logger.Infof("starting pprof at %s", p.Cfg.Addr)
	err := p.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close .
func (p *provider) Close() error {
	if p.server == nil {
		return nil
	}
	err := p.server.Close()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func init() {
	servicehub.RegisterProvider("pprof", &providerDefine{})
}
