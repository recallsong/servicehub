package httpserver

import (
	"net/http"

	"github.com/go-playground/validator"
	"github.com/labstack/echo"
	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/servicehub"
)

// config .
type config struct {
	Addr            string `json:"addr"`
	PrintRoutes     bool   `json:"print_routes"`
	IndexShowRoutes bool   `json:"index_show_routes"`
}

type provider struct {
	cfg    config
	logger logs.Logger
	server *echo.Echo
	router *router
}

func newProvider() servicehub.ServiceProvider {
	p := &provider{
		cfg: config{
			Addr:        ":8080",
			PrintRoutes: true,
		},
		router: &router{
			routeMap: make(map[routeKey]*route),
		},
	}
	p.router.p = p
	return p
}

// Name .
func (p *provider) Name() string { return "http-server" }

// Services .
func (p *provider) Services() []string {
	return []string{"http-server", "api-server"}
}

// config .
func (p *provider) Config() interface{} { return &p.cfg }

// SetLogger .
func (p *provider) SetLogger(logger logs.Logger) {
	p.logger = logger
}

// Init .
func (p *provider) Init(ctx servicehub.Context) error {
	p.server = echo.New()
	p.server.HideBanner = true
	p.server.HidePort = true
	p.server.Binder = &dataBinder{}
	p.server.Validator = &structValidator{validator: validator.New()}
	p.server.Use(func(fn echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			ctx = &context{Context: ctx}
			err := fn(ctx)
			if err != nil {
				p.logger.Error(err)
				return nil
			}
			return nil
		}
	})
	return nil
}

// Start .
func (p *provider) Start() error {
	if p.cfg.PrintRoutes || p.cfg.IndexShowRoutes {
		p.router.Normalize()
	}
	if p.cfg.PrintRoutes {
		for _, route := range p.router.routes {
			if !route.hide {
				p.logger.Infof("--> %s", route.String())
			}
		}
	}
	p.logger.Infof("starting http server at %s", p.cfg.Addr)
	err := p.server.Start(p.cfg.Addr)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close .
func (p *provider) Close() error {
	if p.server == nil || p.server.Server == nil {
		return nil
	}
	err := p.server.Server.Close()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Provide .
func (p *provider) Provide(name string, args ...interface{}) interface{} {
	var intercepters []Intercepter
	for _, arg := range args {
		inter, ok := arg.(Intercepter)
		if ok {
			intercepters = append(intercepters, inter)
			continue
		}
		inter, ok = arg.(func(handler func(ctx Context) error) func(ctx Context) error)
		if ok {
			intercepters = append(intercepters, inter)
			continue
		}
	}
	return Router(&router{
		p:            p,
		routeMap:     p.router.routeMap,
		group:        name,
		intercepters: intercepters,
	})
}

func init() {
	servicehub.RegisterProvider("http-server", newProvider)
}
