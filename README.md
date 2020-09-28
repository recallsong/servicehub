# servicehub
服务管理器 *servicehub.Hub*，管理服务的启动、初始化、依赖关系、关闭等

实现 *servicehub.Provider* 接口来提供服务。

*github.com/recallsong/servicehub-providers* 里已经提供了一些比较有用的 Provider，可直接拿来使用。

## 例子
配置文件 *examples.yaml*
```yaml
hello:
    message: "hello world"
```
*main.go*
```go
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
```
[例子详情](./examples/main.go)

## 配置读取
支持以下方式获取配置，读取优先级由低到高分别为：
* default Tag In Struct
* System Environment Variable
* .env File Environment Variable
* Config File
* Flag

支持的配置文件格式：
* yaml、yml
* json
* hcl
* toml
* ...

## TODO List
* CLI tools to quick start
* Test Case

## License
[Licensed under MIT](./LICENSE)