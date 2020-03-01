package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/servicehub"
)

// Config .
type Config struct {
	Addr string `json:"addr"`
}

// Provider .
type Provider struct {
	cfg    Config
	logger logs.Logger
	server *http.Server
}

// New .
func New() servicehub.ServiceProvider {
	return &Provider{
		cfg: Config{
			Addr: ":6580",
		},
	}
}

// Name .
func (p *Provider) Name() string { return "pprof" }

// Services .
func (p *Provider) Services() []string { return []string{"pprof"} }

// Config .
func (p *Provider) Config() interface{} { return &p.cfg }

// SetLogger .
func (p *Provider) SetLogger(logger logs.Logger) {
	p.logger = logger
}

// Init .
func (p *Provider) Init(ctx servicehub.Context) error {
	server := &http.Server{Addr: p.cfg.Addr}
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
func (p *Provider) Start() error {
	p.logger.Infof("starting pprof at %s", p.cfg.Addr)
	err := p.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close .
func (p *Provider) Close() error {
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
	servicehub.RegisterProvider("pprof", New)
}
