package config

import (
	"bytes"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"reflect"
	"time"
)

// Config represents config for webhooker.
// It contains common settings and rules.
type Config struct {
	BlockCacheSize              int                               `mapstructure:"block_cache_size"`
	PoolSize                    int                               `mapstructure:"pool_size"`
	Runners                     int                               `mapstructure:"runners"`
	RemoteConfigRefreshInterval time.Duration                     `mapstructure:"remote_config_refresh_interval"`
	CommonParameters            map[string]map[string]interface{} `mapstructure:"common_parameters"`
	Rules                       model.Rules                       `mapstructure:"rules"`
}

const (
	defaultConfigType     = "yaml"
	defaultBlockCacheSize = 50 * 1024 * 1024 // 50 MB
	defaultPoolSize       = 100
	defaultRunners        = 10

	// ProviderFile constant represents correct string value of program parameter.
	ProviderFile = "file"

	context = "startup"
)

// New creates Config instance.
func New(
	readFileFunc func(string) ([]byte, error),
	configer configer,
	provider, rawPath string,
	logger *logrus.Logger,
	taskExecutors map[string]executor.TaskExecutor,
	refreshIterations int,
) (*Config, error) {

	endpoint, path, extension, err := utils.ParsePath(rawPath, defaultConfigType, provider)
	if err != nil {
		return nil, err
	}

	configer.SetConfigType(extension)

	err = readConfig(readFileFunc, configer, provider, endpoint, path)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = configer.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	err = conf.prepare(taskExecutors)
	if err != nil {
		return nil, err
	}

	if conf.RemoteConfigRefreshInterval > 0 && provider != ProviderFile {
		go refreshDaemon(conf, provider, path, configer, logger, taskExecutors, refreshIterations)
	}

	return conf, nil
}

func readConfig(readFileFunc func(string) ([]byte, error), configer configer, provider, endpoint, path string) (err error) {
	if provider == ProviderFile {
		if len(endpoint) != 0 {
			return fmt.Errorf("incorrect path for provider %v", provider)
		}

		var b []byte
		b, err = readFileFunc(path)
		if err != nil {
			return
		}

		return configer.ReadConfig(bytes.NewReader(b))
	}

	if len(endpoint) == 0 {
		return fmt.Errorf("empty endpoint for provider %v", provider)
	}

	err = configer.AddRemoteProvider(provider, endpoint, path)
	if err != nil {
		return
	}

	return configer.ReadRemoteConfig()
}

func refreshDaemon(config *Config, provider, path string, configer configer, logger *logrus.Logger, taskExecutors map[string]executor.TaskExecutor, refreshIterations int) {
	i := 1

	ctxLogger := logger.WithFields(logrus.Fields{
		"context": context,
		"params": map[string]interface{}{
			"configProvider": provider,
			"configPath":     path,
		},
		"config": *config,
	})

	var (
		changed bool
		err     error
	)
	for {
		time.Sleep(config.RemoteConfigRefreshInterval)

		ctxLogger = ctxLogger.WithField("iteration", i)
		ctxLogger.Debug("starts refreshing config")

		changed, err = refresh(config, configer, taskExecutors)

		ctxLogger = ctxLogger.WithField("config", *config)
		if err != nil {
			ctxLogger.Errorf("config refresh error: %v", err)
		} else {
			if changed {
				ctxLogger.Info("successfully done refreshing config: config changed")
			} else {
				ctxLogger.Debug("successfully done refreshing config: no changes")
			}
		}

		if refreshIterations > 0 && i >= refreshIterations {
			// testing purposes only
			return
		}
		i++
	}
}

func refresh(currConfig *Config, configer configer, taskExecutors map[string]executor.TaskExecutor) (changed bool, err error) {
	err = configer.WatchRemoteConfig()
	if err != nil {
		return
	}

	newConfig := &Config{}
	err = configer.Unmarshal(newConfig)
	if err != nil {
		return
	}

	err = newConfig.prepare(taskExecutors)
	if err != nil {
		return
	}

	if !reflect.DeepEqual(newConfig.Rules, currConfig.Rules) {
		// we can't replace config completely because handler and runners depends on rules pointer
		err = copier.Copy(&currConfig.Rules, &newConfig.Rules)
		changed = true
	}

	if !reflect.DeepEqual(currConfig.CommonParameters, newConfig.CommonParameters) {
		currConfig.CommonParameters = newConfig.CommonParameters
		changed = true
	}

	if currConfig.RemoteConfigRefreshInterval != newConfig.RemoteConfigRefreshInterval {
		currConfig.RemoteConfigRefreshInterval = newConfig.RemoteConfigRefreshInterval
		changed = true
	}

	return
}

func (c *Config) prepare(taskExecutors map[string]executor.TaskExecutor) (err error) {

	// default values
	c.fillDefaults()

	return c.Rules.Prepare(c.CommonParameters, taskExecutors)
}

func (c *Config) fillDefaults() {
	if c.BlockCacheSize <= 0 {
		c.BlockCacheSize = defaultBlockCacheSize
	}

	if c.PoolSize <= 0 {
		c.PoolSize = defaultPoolSize
	}

	if c.Runners <= 0 {
		c.Runners = defaultRunners
	}
}

//go:generate mockgen -source=config.go -destination=config_mocks.go -package=config doc github.com/golang/mock/gomock

type configer interface {
	SetConfigType(in string)
	ReadConfig(in io.Reader) error
	AddRemoteProvider(provider, endpoint, path string) error
	ReadRemoteConfig() error
	Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error
	WatchRemoteConfig() error
}
