package servicehub

import (
	"fmt"
	"os"
)

var globalProviders = make(map[string]ProviderDefine)

// RegisterGlobalSpec .
func RegisterGlobalSpec(name string, spec *Spec) {
	RegisterGlobalProvider(name, &specDefine{spec})
}

// RegisterGlobalProvider .
func RegisterGlobalProvider(name string, define ProviderDefine) {
	if _, ok := globalProviders[name]; ok {
		fmt.Printf("global provider %s already exist\n", name)
		os.Exit(-1)
	}
	globalProviders[name] = define
}
