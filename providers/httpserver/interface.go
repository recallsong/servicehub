package httpserver

import (
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/labstack/echo"
)

// Router .
type Router interface {
	GET(path string, handler interface{}, options ...interface{})
	POST(path string, handler interface{}, options ...interface{})
	DELETE(path string, handler interface{}, options ...interface{})
	PUT(path string, handler interface{}, options ...interface{})
	PATCH(path string, handler interface{}, options ...interface{})
	HEAD(path string, handler interface{}, options ...interface{})
	CONNECT(path string, handler interface{}, options ...interface{})
	OPTIONS(path string, handler interface{}, options ...interface{})
	TRACE(path string, handler interface{}, options ...interface{})

	Any(path string, handler interface{}, options ...interface{})
	Static(prefix, root string, options ...interface{})
	File(path, filepath string, options ...interface{})
	// StaticFS(prefix string, fs http.FileSystem, options ...interface{})

	Add(method, path string, handler interface{}, options ...interface{})
}

// Context handler context.
type Context interface {
	SetAttribute(key string, val interface{})
	Attribute(key string) interface{}
	Attributes() map[string]interface{}
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	Param(name string) string
	ParamNames() []string
}

// Intercepter .
type Intercepter func(handler func(ctx Context) error) func(ctx Context) error

type context struct {
	echo.Context
	data map[string]interface{}
}

func (c context) SetAttribute(key string, val interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = val
}

func (c context) Attribute(key string) interface{} {
	if c.data == nil {
		return nil
	}
	return c.data[key]
}

func (c context) Attributes() map[string]interface{} {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	return c.data
}

func (c context) ResponseWriter() http.ResponseWriter {
	return c.Context.Response()
}

type routeKey struct {
	method string
	path   string
}

type route struct {
	*routeKey
	group string
	hide  bool
	desc  string
}

func (r *route) String() string {
	return fmt.Sprintf("[%s] %-7s %s", r.group, r.method, r.path)
}

type router struct {
	p            *provider
	routeMap     map[routeKey]*route
	routes       []*route
	group        string
	intercepters []Intercepter
}

func (r *router) Server() *echo.Echo {
	return r.p.server
}

func (r *router) Add(method, path string, handler interface{}, options ...interface{}) {
	method = strings.ToUpper(method)
	key := routeKey{
		method: method,
		path:   path,
	}
	if rt, ok := r.routeMap[key]; ok {
		if rt.group != r.group {
			panic(fmt.Errorf("httpserver routes [%s %s] conflict between groups (%s, %s)",
				key.method, key.path, rt.group, r.group))
		} else {
			panic(fmt.Errorf("httpserver routes [%s %s] conflict in group %s",
				key.method, key.path, rt.group))
		}
	}
	route := &route{
		routeKey: &key,
		group:    r.group,
	}
	for _, opt := range options {
		processOptions(route, opt)
	}
	r.routeMap[key] = route
	r.routes = append(r.routes, route)

	if handler != nil {
		r.add(method, path, handler)
	}
}

type option func(r *route)

func processOptions(r *route, opt interface{}) {
	if fn, ok := opt.(option); ok {
		fn(r)
	}
}

// WithDescription .
func WithDescription(desc string) interface{} {
	return option(func(r *route) {
		r.desc = desc
	})
}

// WithHide .
func WithHide(hide bool) interface{} {
	return option(func(r *route) {
		r.hide = hide
	})
}

type routesSorter []*route

func (s routesSorter) Len() int {
	return len(s)
}

func (s routesSorter) Less(i, j int) bool {
	if s[i].group == s[j].group {
		if s[i].path == s[j].path {
			return s[i].method < s[j].method
		}
		return s[i].path < s[j].path
	}
	return s[i].group < s[j].group
}

func (s routesSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (r *router) Normalize() {
	r.routes = nil
	for _, route := range r.routeMap {
		r.routes = append(r.routes, route)
	}
	sort.Sort(routesSorter(r.routes))
}

func (r *router) GET(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodGet, path, handler, options...)
}

func (r *router) POST(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodPost, path, handler, options...)
}

func (r *router) DELETE(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodDelete, path, handler, options...)
}

func (r *router) PUT(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodPut, path, handler, options...)
}

func (r *router) PATCH(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodPatch, path, handler, options...)
}

func (r *router) HEAD(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodHead, path, handler, options...)
}

func (r *router) CONNECT(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodConnect, path, handler, options...)
}

func (r *router) OPTIONS(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodOptions, path, handler, options...)
}

func (r *router) TRACE(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodTrace, path, handler, options...)
}

func (r *router) Any(path string, handler interface{}, options ...interface{}) {
	r.Add(http.MethodConnect, path, handler, options...)
	r.Add(http.MethodDelete, path, handler, options...)
	r.Add(http.MethodGet, path, handler, options...)
	r.Add(http.MethodHead, path, handler, options...)
	r.Add(http.MethodOptions, path, handler, options...)
	r.Add(http.MethodPatch, path, handler, options...)
	r.Add(http.MethodPost, path, handler, options...)
	r.Add(http.MethodPut, path, handler, options...)
	r.Add(http.MethodTrace, path, handler, options...)
}

func (r *router) Static(prefix, root string, options ...interface{}) {
	r.Add(http.MethodGet, prefix+"/**", nil, options...)
	r.p.server.Static(prefix, root)
	r.p.server.File("/", path.Join(prefix, "index.html"))
}

func (r *router) File(path, filepath string, options ...interface{}) {
	r.Add(http.MethodGet, path, nil, options...)
	r.p.server.File(path, filepath)
}

// func (r *router) Static(prefix, root string, options ...interface{}) {
// 	r.Add(http.MethodGet, prefix+"/**", nil, options...)
// 	r.p.server.Static(prefix, root)
// 	r.p.server.File("/", path.Join(prefix, "index.html"))
// }