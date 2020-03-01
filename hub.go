package servicehub

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/recallsong/go-utils/errorx"
	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/go-utils/logs/stdout"
	"github.com/recallsong/go-utils/os/signalx"
	graph "github.com/recallsong/servicehub/dependency-graph"
	"github.com/spf13/viper"
)

// Hub .
type Hub struct {
	logger       logs.Logger
	providersMap map[string][]*providerContext
	providers    []*providerContext
	lock         sync.RWMutex

	started bool
	wg      sync.WaitGroup

	// options
	requires   []string
	autoCreate bool
}

// New .
func New(options ...interface{}) *Hub {
	hub := &Hub{}
	for _, opt := range options {
		processOptions(hub, opt)
	}
	if hub.logger == nil {
		hub.logger = &stdout.Stdout{}
	}
	return hub
}

// Init .
func (h *Hub) Init(viper *viper.Viper) (err error) {
	defer func() {
		exp := recover()
		if exp != nil {
			if e, ok := exp.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", exp)
			}
		}
		if err != nil {
			h.logger.Errorf("fail to init service hub: %s", err)
		}
	}()

	err = h.loadProviders(viper)
	if err != nil {
		return err
	}

	// check requires
	for _, item := range h.requires {
		if _, ok := h.providersMap[item]; !ok {
			if !h.autoCreate {
				return fmt.Errorf("provider %s is required", item)
			}
			err = h.addProvider(item, nil)
			if err != nil {
				return err
			}
		}
	}

	resolved, err := h.resolveDependency(h.providersMap)
	if err != nil {
		return fmt.Errorf("fail to resolve dependency: %s", err)
	}
	h.providers = resolved

	for _, ctx := range h.providers {
		err = ctx.Init()
		if err != nil {
			return err
		}
		if len(ctx.Dependencies()) > 0 {
			h.logger.Infof("provider %s (depends %v) initialized", ctx.provider.Name(), ctx.Dependencies())
		} else {
			h.logger.Infof("provider %s initialized", ctx.provider.Name())
		}
	}
	return nil
}

func (h *Hub) resolveDependency(providersMap map[string][]*providerContext) ([]*providerContext, error) {
	services := map[string][]*providerContext{}
	for _, p := range providersMap {
		service := p[0].provider.Services()
		for _, s := range service {
			if exist, ok := services[s]; ok {
				return nil, fmt.Errorf("service %s conflict between %s and %s", s, exist[0].provider.Name(), p[0].provider.Name())
			}
			services[s] = p
		}
	}
	var depGraph graph.Graph
	for name, p := range providersMap {
		depends := p[0].Dependencies()
		providers := map[string]*providerContext{}
		for _, service := range depends {
			if deps, ok := services[service]; ok || len(deps) <= 0 {
				if len(deps) > 1 {
					return nil, fmt.Errorf("provider %s ambiguity for service %s", deps[0].provider.Name(), service)
				}
				providers[deps[0].provider.Name()] = deps[0]
			} else {
				return nil, fmt.Errorf("miss provider of service %s", service)
			}
		}
		node := graph.NewNode(name)
		for dep := range providers {
			node.Deps = append(node.Deps, dep)
		}
		depGraph = append(depGraph, node)
	}
	resolved, err := graph.Resolve(depGraph)
	if err != nil {
		depGraph.Display()
		return nil, err
	}
	var providers []*providerContext
	for _, node := range resolved {
		for _, provider := range providersMap[node.Name] {
			providers = append(providers, provider)
		}
	}
	return providers, nil
}

// StartWithSignal .
func (h *Hub) StartWithSignal() error {
	sigs := []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
	h.logger.Info("signals to quit:", sigs)
	return h.Start(signalx.Notify(sigs...))
}

// Start .
func (h *Hub) Start(closer ...<-chan os.Signal) (err error) {
	h.lock.Lock()
	num := len(h.providers)
	ch := make(chan error, num)
	h.wg.Add(num)
	for _, item := range h.providers {
		go func(key string, provider ServiceProvider) {
			err := provider.Start()
			if key != provider.Name() {
				key = fmt.Sprintf("%s (%s)", key, provider.Name())
			}
			if err != nil {
				h.logger.Errorf("fail to exit provider %s: %s", key, err)
			} else {
				h.logger.Infof("provider %s exit", key)
			}
			h.wg.Done()
			ch <- err
		}(item.key, item.provider)
	}
	h.started = true
	h.lock.Unlock()
	runtime.Gosched()

	for _, ch := range closer {
		go func(ch <-chan os.Signal) {
			select {
			case <-ch:
				fmt.Println()
				wait := make(chan error)
				go func() {
					wait <- h.Close()
				}()
				select {
				case <-time.After(10 * time.Second):
					h.logger.Errorf("exit service manager timeout !")
					os.Exit(1)
				case err := <-wait:
					if err != nil {
						h.logger.Errorf("fail to exit: %s", err)
						os.Exit(1)
					}
				}
			}
		}(ch)
	}
	// wait to stop
	errs := errorx.Errors{}
	for i := 0; i < num; i++ {
		select {
		case err := <-ch:
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.MaybeUnwrap()
}

// Close .
func (h *Hub) Close() error {
	h.lock.Lock()
	if !h.started {
		h.lock.Unlock()
		return nil
	}
	var errs errorx.Errors
	for i := len(h.providers) - 1; i >= 0; i-- {
		err := h.providers[i].provider.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	h.wg.Wait()
	h.started = false
	h.lock.Unlock()
	return errs.MaybeUnwrap()
}

type providerContext struct {
	hub      *Hub
	key      string
	cfg      interface{}
	provider ServiceProvider
}

func (c *providerContext) Init() (err error) {
	if configer, ok := c.provider.(ServiceConfigurator); ok {
		cfg := configer.Config()
		if c.cfg != nil {
			err := unmarshalConfig(c.cfg, cfg)
			if err != nil {
				return fmt.Errorf("fail to Unmarshal provider %s config: %s", c.provider.Name(), err)
			}
		}
		c.cfg = cfg
	}
	if setter, ok := c.provider.(ServiceLogger); ok {
		setter.SetLogger(c.Logger())
	}
	if initializer, ok := c.provider.(ServiceInitializer); ok {
		err = initializer.Init(c)
		if err != nil {
			return fmt.Errorf("fail to Init provider %s: %s", c.provider.Name(), err)
		}
	}
	return nil
}

// Dependencies .
func (c *providerContext) Dependencies() []string {
	if deps, ok := c.provider.(ServiceDependencies); ok {
		return deps.Dependencies()
	}
	return nil
}

// Logger .
func (c *providerContext) Logger() logs.Logger {
	if c.hub.logger == nil {
		return nil
	}
	return c.hub.logger.Sub(c.provider.Name())
}

// Config .
func (c *providerContext) Config() interface{} {
	return c.cfg
}

// Hub .
func (c *providerContext) Hub() *Hub {
	return c.hub
}

// Provider .
func (c *providerContext) Provider(name string, options ...interface{}) interface{} {
	if providers, ok := c.hub.providersMap[name]; ok {
		if len(providers) > 0 {
			provider := providers[0].provider
			if prod, ok := provider.(DependencyProvider); ok {
				return prod.Provide(c.provider.Name(), options...)
			}
			return provider
		}
	}
	return nil
}

// Run .
func (h *Hub) Run(viper *viper.Viper) {
	err := h.Init(viper)
	defer func() {
		if err != nil {
			os.Exit(1)
		}
	}()
	if err != nil {
		return
	}
	defer h.Close()
	err = h.StartWithSignal()
	if err != nil {
		return
	}
}
