package servicehub

import (
	"fmt"
	"reflect"
)

var (
	globalServices     map[string]reflect.Value       = map[string]reflect.Value{}
	globalServiceTypes map[reflect.Type]reflect.Value = map[reflect.Type]reflect.Value{}
)

// RegisterService register global service
func RegisterService(obj interface{}, name string, typs ...reflect.Type) {
	value := reflect.ValueOf(obj)
	objtyp := value.Type()
	if len(name) > 0 {
		if _, ok := globalServices[name]; ok {
			panic(fmt.Errorf("global service %q already exist", name))
		}
		globalServices[name] = value
	}
	for _, typ := range typs {
		if !objtyp.AssignableTo(typ) {
			panic(fmt.Errorf("global service type %v can't assign to %v", objtyp, typ))
		}
		if _, ok := globalServiceTypes[typ]; ok {
			panic(fmt.Errorf("global service type %q already exist", name))
		}
		globalServiceTypes[typ] = value
	}
}
