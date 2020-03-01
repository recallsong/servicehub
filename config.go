package servicehub

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// GetViper .
func GetViper(name string, paths ...string) *viper.Viper {
	v := viper.GetViper()
	v.SetConfigName(name)
	for _, path := range paths {
		v.AddConfigPath(path)
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("config not fount")
		} else {
			fmt.Println("fail to read config ", err)
		}
		os.Exit(1)
	}
	fmt.Println("using config file:", v.ConfigFileUsed())
	return v
}

func unmarshalConfig(input, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		TagName:          "json",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func (h *Hub) loadProviders(viper *viper.Viper) error {
	h.providersMap = map[string][]*providerContext{}
	err := h.doLoadProviders(viper, "providers")
	if err != nil {
		return err
	}
	list := viper.Get("providers")
	if list != nil {
		switch providers := list.(type) {
		case []interface{}:
			for _, item := range providers {
				if cfg, ok := item.(map[string]interface{}); ok {
					err = h.addProvider("", cfg)
					if err != nil {
						return nil
					}
				} else {
					return fmt.Errorf("invalid provider config type: %v", reflect.TypeOf(cfg))
				}
			}
		case map[string]interface{}:
			err = h.doLoadProviders(viper.Sub("providers"), "")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Hub) doLoadProviders(viper *viper.Viper, filter string) error {
	keys := viper.AllSettings()
	for key, cfg := range keys {
		key = strings.ReplaceAll(key, "_", "-")
		if key == filter {
			continue
		}
		err := h.addProvider(key, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Hub) addProvider(key string, cfg interface{}) error {
	name := key
	if cfg != nil {
		if v, ok := cfg.(map[string]interface{}); ok {
			if val, ok := v["name"]; ok {
				if n, ok := val.(string); ok {
					name = n
				}
			}
			if val, ok := v["enable"]; ok {
				if enable, ok := val.(bool); ok && !enable {
					return nil
				}
			}
		}
	}
	if name == "" {
		return fmt.Errorf("provider name must not be empty")
	}
	creator, ok := ServiceProviders[name]
	if !ok {
		return fmt.Errorf("provider %s not exist", name)
	}
	provider := creator()
	h.providersMap[name] = append(h.providersMap[name], &providerContext{h, key, cfg, provider})
	return nil
}
