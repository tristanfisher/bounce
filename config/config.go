package config

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/spf13/viper"
)

// Config defaults to production values
type Config struct {
	LogLevel         string        `default:"error"`
	ShutdownDeadline time.Duration `default:"10s"`

	HttpServerName      string        `default:"localhost.localdomain"`
	HttpServerAddr      string        `default:":80"`
	HttpServerKeepAlive bool          `default:"true"`
	HttpReadTimeout     time.Duration `default:"10s"`
	HttpWriteTimeout    time.Duration `default:"10s"`
	HttpIdleTimeout     time.Duration `default:"30s"`

	HttpsServerName      string `default:"localhost.localdomain"`
	HttpsServerAddr      string `default:":443"`
	HttpsCertificatePath string `default:""`
	HttpsKeyPath         string `default:""`
	HttpsIsOffloaded     bool   `default:"true"`

	DestinationHost string `default:"example.org"`
}

func getDefault(fieldName string, t reflect.Type) (interface{}, error) {
	var i interface{}
	f, ok := t.FieldByName(fieldName)
	if !ok {
		return i, errors.New("missing fieldName: " + fieldName)
	}

	switch f.Type {
	case reflect.TypeOf(time.Duration(0)):
		d, err := time.ParseDuration(f.Tag.Get("default"))
		if err != nil {
			return d, err
		}
		return d, nil
	}

	return f.Tag.Get("default"), nil
}

func New(configFilePath string) (*Config, error) {

	var c Config
	viperConf := viper.New()

	ct := reflect.TypeOf(c)
	for i := 0; i < ct.NumField(); i++ {
		defaultVal, err := getDefault(ct.Field(i).Name, ct)
		if err != nil {
			return nil, fmt.Errorf("could not parse default value.  reason: %s", err)
		}
		viperConf.SetDefault(ct.Field(i).Name, defaultVal)
	}
	if configFilePath != "" {
		viperConf.SetConfigFile(configFilePath)
		err := viperConf.ReadInConfig()
		if err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				return nil, fmt.Errorf("could not read specified file.  reason: %s", err)
			}
			return nil, fmt.Errorf("failed to load config json (try passing through a linter).  reason: %s", err)
		}
	}

	viperConf.AutomaticEnv()

	err := viperConf.Unmarshal(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
