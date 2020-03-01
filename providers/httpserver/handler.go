package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/go-playground/validator"
	"github.com/labstack/echo"
	"github.com/recallsong/go-utils/errorx"
	"github.com/recallsong/go-utils/reflectx"
)

func (r *router) add(method, path string, handler interface{}) {
	var echoHandler echo.HandlerFunc
	switch fn := handler.(type) {
	case echo.HandlerFunc:
		echoHandler = fn
	case func(echo.Context) error:
		echoHandler = echo.HandlerFunc(fn)
	case func(echo.Context):
		echoHandler = echo.HandlerFunc(func(ctx echo.Context) error {
			fn(ctx)
			return nil
		})
	case http.HandlerFunc:
		echoHandler = echo.HandlerFunc(func(ctx echo.Context) error {
			fn(ctx.Response(), ctx.Request())
			return nil
		})
	case func(http.ResponseWriter, *http.Request):
		echoHandler = echo.HandlerFunc(func(ctx echo.Context) error {
			fn(ctx.Response(), ctx.Request())
			return nil
		})
	case func(*http.Request, http.ResponseWriter):
		echoHandler = echo.HandlerFunc(func(ctx echo.Context) error {
			fn(ctx.Request(), ctx.Response())
			return nil
		})
	case http.Handler:
		echoHandler = echo.HandlerFunc(func(ctx echo.Context) error {
			fn.ServeHTTP(ctx.Response(), ctx.Request())
			return nil
		})
	default:
		echoHandler = r.handlerWrap(handler)
		if echoHandler == nil {
			panic(fmt.Errorf("%s %s: not support http server handler type: %v", method, path, handler))
		}
	}
	if len(r.intercepters) > 0 {
		originalHandler := echoHandler
		handler := func(ctx Context) error {
			c := ctx.(*context)
			return originalHandler(c)
		}
		for i := len(r.intercepters) - 1; i >= 0; i-- {
			handler = r.intercepters[i](handler)
		}
		echoHandler = func(ctx echo.Context) error {
			c := ctx.(*context)
			return handler(c)
		}
	}
	r.p.server.Add(method, path, echoHandler)
}

var (
	readerType      = reflect.TypeOf((*io.Reader)(nil)).Elem()
	readCloserType  = reflect.TypeOf((*io.ReadCloser)(nil)).Elem()
	errorType       = reflect.TypeOf((*error)(nil)).Elem()
	requestType     = reflect.TypeOf((*http.Request)(nil))
	responseType    = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
	echoContextType = reflect.TypeOf((*echo.Context)(nil)).Elem()
	contextType     = reflect.TypeOf((*Context)(nil)).Elem()
	interfaceType   = reflect.TypeOf((*interface{})(nil)).Elem()
)

func (r *router) handlerWrap(handler interface{}) echo.HandlerFunc {
	typ := reflect.TypeOf(handler)
	if typ.Kind() == reflect.Func {
		val := reflect.ValueOf(handler)
		var argGets []func(ctx echo.Context) (interface{}, error)
		argNum := typ.NumIn()
		for i := 0; i < argNum; i++ {
			argTyp := typ.In(i)
			getter := argGetter(argTyp)
			if getter == nil {
				return nil
			}
			argGets = append(argGets, getter)
		}
		retNum := typ.NumOut()
		if retNum > 3 {
			return nil
		}
		var retGet func(values []reflect.Value) (*int, io.ReadCloser, io.Reader, interface{}, error)
		var retIndex [5]*int
		var hasRet bool
		for i := 0; i < retNum; i++ {
			retTyp := typ.Out(i)
			index := i
			if retTyp.Kind() == reflect.Int {
				if retIndex[0] == nil {
					retIndex[0] = &index
					hasRet = true
					continue
				}
			} else if retTyp.AssignableTo(readCloserType) {
				if retIndex[1] == nil {
					retIndex[1] = &index
					hasRet = true
					continue
				}
			} else if retTyp.AssignableTo(readerType) {
				if retIndex[2] == nil {
					retIndex[2] = &index
					hasRet = true
					continue
				}
			} else if retTyp == errorType {
				if retIndex[3] == nil {
					retIndex[3] = &index
					hasRet = true
					continue
				}
			} else if retTyp == interfaceType {
				if retIndex[4] == nil {
					retIndex[4] = &index
					hasRet = true
					continue
				}
			}
			return nil
		}
		if hasRet {
			retGet = func(values []reflect.Value) (status *int, readerCloser io.ReadCloser, reader io.Reader, data interface{}, err error) {
				if retIndex[0] != nil {
					val := int(values[*retIndex[0]].Int())
					status = &val
				}
				if retIndex[1] != nil {
					val := values[*retIndex[1]].Interface()
					readerCloser = val.(io.ReadCloser)
				}
				if retIndex[2] != nil {
					val := values[*retIndex[2]].Interface()
					reader = val.(io.Reader)
				}
				if retIndex[3] != nil {
					val := values[*retIndex[3]].Interface()
					if val != nil {
						err = val.(error)
					}
				}
				if retIndex[4] != nil {
					data = values[*retIndex[4]].Interface()
				}
				return
			}
		}
		return echo.HandlerFunc(func(ctx echo.Context) error {
			var values []reflect.Value
			for _, getter := range argGets {
				val, err := getter(ctx)
				if err != nil {
					return err
				}
				value := reflect.ValueOf(val)
				values = append(values, value)
			}
			returns := val.Call(values)
			if retGet == nil {
				return nil
			}
			status, readCloser, reader, data, err := retGet(returns)
			if status != nil {
				ctx.Response().WriteHeader(*status)
			}
			var errs errorx.Errors
			if err != nil {
				errs = append(errs, err)
			}
			if readCloser != nil {
				defer readCloser.Close()
				_, err = io.Copy(ctx.Response(), readCloser)
				if err != nil {
					errs = append(errs, err)
				}
			} else if reader != nil {
				_, err = io.Copy(ctx.Response(), reader)
				if err != nil {
					errs = append(errs, err)
				}
			} else if data != nil {
				err = json.NewEncoder(ctx.Response()).Encode(data)
				if err != nil {
					errs = append(errs, err)
				}
			}
			return errs.MaybeUnwrap()
		})
	}
	return nil
}

func argGetter(argTyp reflect.Type) func(ctx echo.Context) (interface{}, error) {
	if argTyp == requestType {
		return requestGetter
	} else if argTyp == responseType {
		return responseGetter
	} else if argTyp == contextType || argTyp == echoContextType {
		return contextGetter
	} else {
		kind := argTyp.Kind()
		if kind == reflect.String {
			return responseBodyStirngGetter
		} else if kind == reflect.Slice && argTyp.Elem().Kind() == reflect.Uint8 {
			return responseBodyBytesGetter
		}
		typ := argTyp
		for kind == reflect.Ptr {
			typ = typ.Elem()
			kind = typ.Kind()
		}
		switch kind {
		case reflect.Struct:
			var validate bool
			for i, num := 0, typ.NumField(); i < num; i++ {
				if len(typ.Field(i).Tag.Get("validate")) > 0 {
					validate = true
					break
				}
			}
			return responseDataBind(argTyp, validate)
		case reflect.Map, reflect.Interface:
			return responseDataBind(argTyp, false)
		case reflect.String:
			return responseBodyStirngGetter
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.Array, reflect.Slice:
			return responseValuesGetter(argTyp)
		default:
			return nil
		}
	}
}

func requestGetter(ctx echo.Context) (interface{}, error) {
	return ctx.Request(), nil
}
func responseGetter(ctx echo.Context) (interface{}, error) {
	return ctx.Response(), nil
}
func contextGetter(ctx echo.Context) (interface{}, error) {
	return ctx, nil
}
func responseDataBind(typ reflect.Type, validate bool) func(echo.Context) (interface{}, error) {
	return func(ctx echo.Context) (data interface{}, err error) {
		outVal := reflect.New(typ)
		if typ.Kind() != reflect.Ptr {
			data = outVal.Interface()
			err = ctx.Bind(data)
		} else {
			eval := outVal.Elem()
			etype := typ.Elem()
			for etype.Kind() == reflect.Ptr {
				v := reflect.New(etype)
				eval.Set(v)
				eval = v.Elem()
				etype = etype.Elem()
			}
			switch etype.Kind() {
			case reflect.Map:
				v := reflect.New(etype)
				v.Elem().Set(reflect.MakeMap(etype))
				eval.Set(v)
			case reflect.Slice:
				v := reflect.New(etype)
				v.Elem().Set(reflect.MakeSlice(etype, 0, 0))
				eval.Set(v)
			default:
				eval.Set(reflect.New(etype))
			}
			data = eval.Interface()
			err = ctx.Bind(data)
		}
		if err != nil {
			return nil, fmt.Errorf("fail to bind data: %s", err)
		}
		if validate {
			err = ctx.Validate(data)
			if err != nil {
				return nil, fmt.Errorf("fail to validate data: %s", err)
			}
		}
		return outVal.Elem().Interface(), nil
	}
}
func responseValuesGetter(typ reflect.Type) func(ctx echo.Context) (interface{}, error) {
	return func(ctx echo.Context) (interface{}, error) {
		out := reflect.New(typ)
		byts, err := ioutil.ReadAll(ctx.Request().Body)
		if err != nil {
			return nil, fmt.Errorf("fail to read body: %s", err)
		}
		ctx.Request().Body = ioutil.NopCloser(bytes.NewBuffer(byts))
		err = json.Unmarshal(byts, out.Interface())
		if err != nil {
			return nil, fmt.Errorf("fail to Unmarshal body: %s", err)
		}
		return out.Elem().Interface(), nil
	}
}
func responseBodyBytesGetter(ctx echo.Context) (interface{}, error) {
	byts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read body: %s", err)
	}
	ctx.Request().Body = ioutil.NopCloser(bytes.NewBuffer(byts))
	return byts, nil
}

func responseBodyStirngGetter(ctx echo.Context) (interface{}, error) {
	byts, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		return "", fmt.Errorf("fail to read body: %s", err)
	}
	return reflectx.BytesToString(byts), nil
}

type structValidator struct {
	validator *validator.Validate
}

// Validate .
func (v *structValidator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}