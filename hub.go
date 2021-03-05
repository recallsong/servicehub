package servicehub

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/recallsong/go-utils/config"
	"github.com/recallsong/go-utils/encoding/jsonx"
	"github.com/recallsong/go-utils/errorx"
	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/go-utils/logs/golog"
	"github.com/recallsong/go-utils/os/signalx"
	graph "github.com/recallsong/servicehub/dependency-graph"
	"github.com/recallsong/unmarshal"
	unmarshalflag "github.com/recallsong/unmarshal/unmarshal-flag"
	"github.com/spf13/pflag"
)

// Hub .
type Hub struct {
	logger       logs.Logger
	providersMap map[string][]*providerContext
	providers    []*providerContext
	servicesMap  map[string][]*providerContext
	lock         sync.RWMutex

	started bool
	wg      sync.WaitGroup

	listeners []Listener
}

// New .
func New(options ...interface{}) *Hub {
	hub := &Hub{}
	for _, opt := range options {
		processOptions(hub, opt)
	}
	if hub.logger == nil {
		hub.logger = golog.New()
	}
	return hub
}

// Init .
func (h *Hub) Init(config map[string]interface{}, flags *pflag.FlagSet, args []string) (err error) {
	defer func() {
		// exp := recover()
		// if exp != nil {
		// 	if e, ok := exp.(error); ok {
		// 		err = e
		// 	} else {
		// 		err = fmt.Errorf("%v", exp)
		// 	}
		// }
		if err != nil {
			h.logger.Errorf("fail to init service hub: %s", err)
		}
	}()
	for i, l := 0, len(h.listeners); i < l; i++ {
		err = h.listeners[i].BeforeInitialization(h, config)
		if err != nil {
			return err
		}
	}
	err = h.loadProviders(config)
	if err != nil {
		return err
	}

	depGraph, err := h.resolveDependency(h.providersMap)
	if err != nil {
		return fmt.Errorf("fail to resolve dependency: %s", err)
	}

	flags.BoolP("providers", "p", false, "print all providers supported")
	flags.BoolP("graph", "g", false, "print providers dependency graph")
	for _, ctx := range h.providers {
		err = ctx.BindConfig(flags)
		if err != nil {
			return fmt.Errorf("fail to bind config for provider %s: %s", ctx.name, err)
		}
	}
	err = flags.Parse(args)
	if err != nil {
		return fmt.Errorf("fail to bind flags: %s", err)
	}
	if ok, err := flags.GetBool("providers"); err == nil && ok {
		usage := Usage()
		fmt.Println(usage)
		os.Exit(0)
	}
	if ok, err := flags.GetBool("graph"); err == nil && ok {
		depGraph.Display()
		os.Exit(0)
	}
	for _, ctx := range h.providers {
		err = ctx.Init()
		if err != nil {
			return err
		}
		if len(ctx.Dependencies()) > 0 {
			h.logger.Infof("provider %s (depends %v) initialized", ctx.name, ctx.Dependencies())
		} else {
			h.logger.Infof("provider %s initialized", ctx.name)
		}
	}
	for i, l := 0, len(h.listeners); i < l; i++ {
		err = h.listeners[i].AfterInitialization(h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Hub) resolveDependency(providersMap map[string][]*providerContext) (graph.Graph, error) {
	services := map[string][]*providerContext{}
	for _, p := range providersMap {
		service := p[0].define.Service()
		for _, s := range service {
			if exist, ok := services[s]; ok {
				return nil, fmt.Errorf("service %s conflict between %s and %s", s, exist[0].name, p[0].name)
			}
			services[s] = p
		}
	}
	h.servicesMap = services
	var depGraph graph.Graph
	for name, p := range providersMap {
		depends := p[0].Dependencies()
		providers := map[string]*providerContext{}
	loop:
		for _, service := range depends {
			name := service
			var key string
			idx := strings.Index(service, "@")
			if idx > 0 {
				key = service
				name = service[0:idx]
			}
			if deps, ok := services[name]; ok {
				if len(key) > 0 {
					for _, dep := range deps {
						if dep.key == key {
							providers[dep.name] = dep
							continue loop
						}
					}
				} else if len(deps) > 0 {
					providers[deps[0].name] = deps[0]
					continue loop
				}
			}
			return nil, fmt.Errorf("miss provider of service %s", service)
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
		return depGraph, err
	}
	var providers []*providerContext
	for _, node := range resolved {
		providers = append(providers, providersMap[node.Name]...)
	}
	h.providers = providers
	return resolved, nil
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
	ch := make(chan error, len(h.providers))
	var num int
	for _, item := range h.providers {
		if runner, ok := item.provider.(ProviderRunner); ok {
			num++
			h.wg.Add(1)
			go func(key, name string, provider ProviderRunner) {
				if key != name {
					key = fmt.Sprintf("%s (%s)", key, name)
				}
				h.logger.Debugf("provider %s starting ...", key)
				err := provider.Start()
				if err != nil {
					h.logger.Errorf("fail to exit provider %s: %s", key, err)
				} else {
					h.logger.Infof("provider %s exit", key)
				}
				h.wg.Done()
				ch <- err
			}(item.key, item.name, runner)
		}
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
				case <-time.After(30 * time.Second):
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
		if runner, ok := h.providers[i].provider.(ProviderRunner); ok {
			err := runner.Close()
			if err != nil {
				errs = append(errs, err)
			}
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
	name     string
	cfg      interface{}
	provider Provider
	define   ProviderDefine
}

var loggerType = reflect.TypeOf((*logs.Logger)(nil)).Elem()

func (c *providerContext) BindConfig(flags *pflag.FlagSet) (err error) {
	if creator, ok := c.define.(ConfigCreator); ok {
		cfg := creator.Config()
		err = unmarshal.BindDefault(cfg)
		if err != nil {
			return err
		}
		if c.cfg != nil {
			err = config.ConvertData(c.cfg, cfg, "file")
			if err != nil {
				return err
			}
		}
		err = unmarshal.BindEnv(cfg)
		if err != nil {
			return err
		}
		err = unmarshalflag.BindFlag(flags, cfg)
		if err != nil {
			return err
		}
		c.cfg = cfg
	}
	return nil
}

func (c *providerContext) Init() (err error) {
	value := reflect.ValueOf(c.provider)
	typ := value.Type()
	if typ.Kind() == reflect.Ptr {
		for typ.Kind() == reflect.Ptr {
			value = value.Elem()
			typ = value.Type()
		}
		var (
			cfgValue *reflect.Value
			cfgType  reflect.Type
		)
		if c.cfg != nil {
			value := reflect.ValueOf(c.cfg)
			cfgValue = &value
			cfgType = cfgValue.Type()
		}
		if typ.Kind() == reflect.Struct {
			fields := typ.NumField()
			for i := 0; i < fields; i++ {
				field := typ.Field(i)
				if field.Type == loggerType {
					logger := c.Logger()
					value.Field(i).Set(reflect.ValueOf(logger))
				}
				if cfgValue != nil && field.Type == cfgType {
					value.Field(i).Set(*cfgValue)
				}
			}
		}
	}
	if c.cfg != nil {
		key := c.key
		if key != c.name {
			key = fmt.Sprintf("%s (%s)", key, c.name)
		}
		if os.Getenv("LOG_LEVEL") == "debug" {
			fmt.Printf("provider %s config: \n%s\n", key, jsonx.MarshalAndIndent(c.cfg))
		}
		// c.hub.logger.Debugf("provider %s config: \n%s", key, jsonx.MarshalAndIndent(c.cfg))
	}

	if initializer, ok := c.provider.(ProviderInitializer); ok {
		err = initializer.Init(c)
		if err != nil {
			return fmt.Errorf("fail to Init provider %s: %s", c.name, err)
		}
	}
	return nil
}

// Define .
func (c *providerContext) Define() ProviderDefine {
	return c.define
}

// Define .
func (c *providerContext) Dependencies() []string {
	if deps, ok := c.define.(ServiceDependencies); ok {
		return deps.Dependencies()
	}
	return nil
}

// Hub .
func (c *providerContext) Hub() *Hub {
	return c.hub
}

// Logger .
func (c *providerContext) Logger() logs.Logger {
	if c.hub.logger == nil {
		return nil
	}
	return c.hub.logger.Sub(c.name)
}

// Config .
func (c *providerContext) Config() interface{} {
	return c.cfg
}

// Provider .
func (c *providerContext) Service(name string, options ...interface{}) interface{} {
	return c.hub.getService(c.name, name, options...)
}

// Service .
func (h *Hub) Service(name string, options ...interface{}) interface{} {
	return h.getService("", name, options...)
}

// Service .
func (h *Hub) getService(operator, name string, options ...interface{}) interface{} {
	var key string
	idx := strings.Index(name, "@")
	if idx > 0 {
		key = name
		name = name[0:idx]
	}
	if providers, ok := h.servicesMap[name]; ok {
		if len(providers) > 0 {
			var pc *providerContext
			if len(key) > 0 {
				for _, item := range providers {
					if item.key == key {
						pc = item
						break
					}
				}
			} else {
				for _, item := range providers {
					if item.key == item.name {
						pc = item
						break
					}
				}
				if pc == nil && len(providers) > 0 {
					pc = providers[0]
				}
			}
			if pc == nil {
				return nil
			}
			provider := pc.provider
			if prod, ok := provider.(DependencyProvider); ok {
				return prod.Provide(operator, options...)
			}
			return provider
		}
	}
	return nil
}

// Run .
func (h *Hub) Run(name, cfgfile string, args ...string) {
	h.RunWithDefault(name, cfgfile, nil, args...)
}

// RunWithDefault .
func (h *Hub) RunWithDefault(name, cfgfile string, defcfg map[string]interface{}, args ...string) {
	if len(name) <= 0 {
		name = getAppName(args...)
	}
	config.LoadEnvFile()

	var err error
	defer func() {
		if err != nil {
			os.Exit(1)
		}
	}()

	if len(cfgfile) <= 0 && len(defcfg) <= 0 {
		cfgfile = name + ".yaml"
	}
	cfg, err := h.loadConfigWithArgs(cfgfile, args...)
	if err != nil {
		return
	}
	if defcfg != nil {
		for k, v := range defcfg {
			if _, ok := cfg[k]; !ok {
				cfg[k] = v
			}
		}
	}

	flags := pflag.NewFlagSet(name, pflag.ExitOnError)
	flags.StringP("config", "c", cfgfile, "config file to load providers")
	err = h.Init(cfg, flags, args)
	if err != nil {
		return
	}
	defer h.Close()
	err = h.StartWithSignal()
	if err != nil {
		return
	}
}

func getAppName(args ...string) string {
	if len(args) <= 0 {
		return ""
	}
	name := args[0]
	idx := strings.LastIndex(os.Args[0], "/")
	if idx >= 0 {
		return name[idx+1:]
	}
	return ""
}
