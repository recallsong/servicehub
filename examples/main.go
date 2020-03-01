package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/recallsong/go-utils/logs"
	"github.com/recallsong/go-utils/logs/logrusx"
	"github.com/recallsong/servicehub"
	"github.com/recallsong/servicehub/providers/httpserver"
	_ "github.com/recallsong/servicehub/providers/pprof"
)

type config struct {
	Message string `json:"message" yaml:"message"`
}

type helloProvider struct {
	cfg    config
	logger logs.Logger
}

func (p *helloProvider) Name() string                 { return "hello" }
func (p *helloProvider) Services() []string           { return []string{"hello"} }
func (p *helloProvider) Config() interface{}          { return &p.cfg }
func (p *helloProvider) SetLogger(logger logs.Logger) { p.logger = logger }
func (p *helloProvider) Dependencies() []string       { return []string{"http-server"} }
func (p *helloProvider) Init(ctx servicehub.Context) error {
	// 获取依赖的服务 http-server 服务
	routes := ctx.Provider("http-server",
		// 定义拦截器
		func(handler func(ctx httpserver.Context) error) func(ctx httpserver.Context) error {
			return func(ctx httpserver.Context) error {
				fmt.Println("intercept request", ctx.Request().URL.String())
				return handler(ctx)
			}
		},
	).(httpserver.Router)

	// 请求参数为 http.ResponseWriter, *http.Request
	routes.GET("/hello", func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(p.cfg.Message))
	}, httpserver.WithDescription("this is hello provider"))

	// 请求参数为 结构体指针、返回结构体为 status int, data interface{}, err error
	routes.POST("/hello/simple", func(body *struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}) (status int, data interface{}, err error) {
		return http.StatusCreated, body, nil
	})

	// 请求参数为 结构体，校验 message 字段是否为空
	routes.POST("/hello/struct/:name", func(resp http.ResponseWriter, req *http.Request,
		body struct {
			Name    string `param:"name"`
			Message string `json:"message" form:"message" query:"message" validate:"required"`
		},
	) {
		resp.Write([]byte(fmt.Sprint(body)))
	})

	// 请求参数为 结构体
	routes.POST("/hello/struct/ptr", func(resp http.ResponseWriter, req *http.Request,
		body *struct {
			Name    string `param:"name"`
			Message string `json:"message" form:"message" query:"message" validate:"required"`
		},
	) {
		resp.Write([]byte(fmt.Sprint(body)))
	})

	// 请求参数为 http.ResponseWriter, *http.Request, []byte, []byte 表示请求 Body
	routes.Any("/hello/bytes", func(resp http.ResponseWriter, req *http.Request, byts []byte) {
		resp.Write(byts)
	})

	// 请求参数 http.ResponseWriter, *http.Request, int
	routes.Any("/hello/int", func(resp http.ResponseWriter, req *http.Request, body int) {
		resp.Write([]byte(fmt.Sprint(body)))
	})
	routes.Any("/hello/int/ptr", func(resp http.ResponseWriter, req *http.Request, body *int) {
		resp.Write([]byte(fmt.Sprint(*body)))
	})

	// 请求参数 http.ResponseWriter, *http.Request, map[string]interface{}
	routes.Any("/hello/map", func(resp http.ResponseWriter, req *http.Request, body map[string]interface{}) {
		resp.Write([]byte(fmt.Sprint(body)))
	})
	routes.Any("/hello/map/ptr", func(resp http.ResponseWriter, req *http.Request, body ******map[string]interface{}) {
		resp.Write([]byte(fmt.Sprint(*body)))
	})

	// 请求参数 http.ResponseWriter, *http.Request, []interface{}
	routes.Any("/hello/slice", func(resp http.ResponseWriter, req *http.Request, body []interface{}) {
		resp.Write([]byte(fmt.Sprint(body)))
	})

	// 请求参数 httpserver.Context, string
	routes.POST("/hello/context", func(ctx httpserver.Context, body string) {
		ctx.ResponseWriter().Write([]byte(body))
	})

	// 返回参数 status int, body io.Reader
	routes.GET("/hello/response/body", func(ctx httpserver.Context) (status int, body io.Reader) {
		return http.StatusOK, bytes.NewReader([]byte("hello"))
	})

	return nil
}

func (p *helloProvider) Start() error {
	p.logger.Info("now hello provider is running...")
	return nil
}

func (p *helloProvider) Close() error {
	p.logger.Info("now hello provider is closing...")
	return nil
}

func init() {
	servicehub.RegisterProvider("hello", func() servicehub.ServiceProvider {
		return &helloProvider{
			cfg: config{
				Message: "hello world",
			},
		}
	})
}

func main() {
	hub := servicehub.New(
		servicehub.WithRequiredServices(true, "http-server"),
		servicehub.WithLogger(logrusx.New()),
	)
	hub.Run(servicehub.GetViper("examples", "./"))
}