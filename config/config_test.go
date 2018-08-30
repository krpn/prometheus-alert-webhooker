package config

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"net/url"
	"regexp"
	"testing"
	"time"
)

var (
	yamlConfigBytes = []byte(`
block_cache_size: 104857600
pool_size: 100
runners: 30
remote_config_refresh_interval: 1ns
common_parameters:
  jenkins1:
    endpoint: https://j.company.com/
    login: admin
    password: qwerty123
rules:
- name: LowDiskSpaceFix
  conditions:
    alert_labels:
      alertname: LowDiskSpace
      instance: ^logs_(.*?)
    alert_annotations:
      webhooker_enabled: (.*?)
  actions:
  - executor: shell
    parameters:
      command: ./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}
    block: 10m
- name: AnyAlertFix
  conditions:
    alert_annotations:
      webhooker_job: (.*?)
  actions:
  - executor: jenkins
    common_parameters: jenkins1
    parameters:
      job_name: ${ANNOTATIONS_WEBHOOKER_JOB}
      instance: ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}
    block: 5m
`)
	jsonConfigBytes = []byte(`
{
  "block_cache_size": 104857600,
  "pool_size": 100,
  "runners": 30,
  "remote_config_refresh_interval": "1ns",
  "common_parameters": {
    "jenkins1": {
      "endpoint": "https://j.company.com/",
      "login": "admin",
      "password": "qwerty123"
    }
  },
  "rules": [
    {
      "name": "LowDiskSpaceFix",
      "conditions": {
        "alert_labels": {
          "alertname": "LowDiskSpace",
          "instance": "^logs_(.*?)"
        },
        "alert_annotations": {
          "webhooker_enabled": "(.*?)"
        }
      },
      "actions": [
        {
          "executor": "shell",
          "parameters": {
            "command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"
          },
          "block": "10m"
        }
      ]
    },
    {
      "name": "AnyAlertFix",
      "conditions": {
        "alert_annotations": {
          "webhooker_job": "(.*?)"
        }
      },
      "actions": [
        {
          "executor": "jenkins",
          "common_parameters": "jenkins1",
          "parameters": {
            "job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
            "instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"
          },
          "block": "5m"
        }
      ]
    }
  ]
}
`)
	jsonWithoutRulesConfigBytes = []byte(`
{
  "block_cache_size": 104857600,
  "pool_size": 100,
  "remote_config_refresh_interval": "1ns",
  "runners": 30
}
`)
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configerMock := NewMockconfiger(ctrl)

	executorShellMock := executor.NewMockTaskExecutor(ctrl)
	executorJenkinsMock := executor.NewMockTaskExecutor(ctrl)

	taskExecutors := map[string]executor.TaskExecutor{
		"shell":   executorShellMock,
		"jenkins": executorJenkinsMock,
	}

	type testTableData struct {
		tcase              string
		configBytes        []byte
		configer           configer
		supportedProviders []string
		supportedExts      []string
		configProvider     string
		configPath         string
		refreshIterations  int
		readFileFuncErr    error
		expectFunc         func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T)
		expectedConfig     func() *Config
		expectedErr        error
		expectedLogs       []string
	}

	testTable := []testTableData{
		{
			tcase:              "yaml",
			configBytes:        yamlConfigBytes,
			configer:           viper.New(),
			supportedProviders: SupportedProviders,
			supportedExts:      viper.SupportedExts,
			configProvider:     ProviderFile,
			configPath:         "Config/Config.yaml",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				eShell.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
				}).Return(nil)
				eJenkins.EXPECT().ValidateParameters(
					map[string]interface{}{
						"job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
						"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
						"endpoint": "https://j.company.com/",
						"login":    "admin",
						"password": "qwerty123",
					}).Return(nil)
			},
			expectedConfig: func() *Config { return getExpectedConfigCompiled(taskExecutors) },
			expectedErr:    nil,
			expectedLogs:   []string{},
		},
		{
			tcase:              "json",
			configBytes:        jsonConfigBytes,
			configer:           viper.New(),
			supportedProviders: SupportedProviders,
			supportedExts:      viper.SupportedExts,
			configProvider:     ProviderFile,
			configPath:         "Config/Config.json",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				eShell.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
				}).Return(nil)
				eJenkins.EXPECT().ValidateParameters(
					map[string]interface{}{
						"job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
						"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
						"endpoint": "https://j.company.com/",
						"login":    "admin",
						"password": "qwerty123",
					}).Return(nil)
			},
			expectedConfig: func() *Config { return getExpectedConfigCompiled(taskExecutors) },
			expectedErr:    nil,
			expectedLogs:   []string{},
		},
		{
			tcase:              "json without rules",
			configBytes:        jsonWithoutRulesConfigBytes,
			configer:           viper.New(),
			supportedProviders: SupportedProviders,
			supportedExts:      viper.SupportedExts,
			configProvider:     ProviderFile,
			configPath:         "Config/Config.json",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("empty rules list"),
		},
		{
			tcase:              "read file error",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file", "consul"},
			supportedExts:      []string{"json"},
			configProvider:     ProviderFile,
			configPath:         "Config/Config.json",
			readFileFuncErr:    errors.New("read file error"),
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("read file error"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "read Config error",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file", "consul"},
			supportedExts:      []string{"json"},
			configProvider:     ProviderFile,
			configPath:         "Config/Config.json",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
				c.EXPECT().ReadConfig(bytes.NewReader(configBytes)).Return(errors.New("read Config error"))
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("read Config error"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "unmarshal error",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file"},
			supportedExts:      []string{"json"},
			configProvider:     ProviderFile,
			configPath:         "Config/Config.json",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
				c.EXPECT().ReadConfig(bytes.NewReader(configBytes)).Return(nil)
				c.EXPECT().Unmarshal(&Config{}).Return(errors.New("unmarshal error"))
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("unmarshal error"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "unsupported Config profider",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file"},
			supportedExts:      []string{"json"},
			configProvider:     "zookeeper",
			configPath:         "Config/Config.jpeg",
			refreshIterations:  1,
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("unsupported Config provider zookeeper"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "unsupported Config type",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file"},
			supportedExts:      []string{"json"},
			configProvider:     ProviderFile,
			configPath:         "Config/Config.jpeg",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("unsupported Config type jpeg"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "uncorrect url",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file", "consul"},
			supportedExts:      []string{"json"},
			configProvider:     "consul",
			configPath:         "http://127 0 0 1:4001/config/hugo.json?ver=1",
			refreshIterations:  1,
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    &url.Error{Op: "parse", URL: "http://127 0 0 1:4001/config/hugo.json?ver=1", Err: url.InvalidHostError(" ")},
			expectedLogs:   []string{},
		},
		{
			tcase:              "incorrect path for provider",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: []string{"file"},
			supportedExts:      []string{"json"},
			configProvider:     ProviderFile,
			configPath:         "http://127.0.0.1:4001/config/hugo.json",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("incorrect path for provider file"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "empty endpoint for provider",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: SupportedProviders,
			supportedExts:      []string{"json"},
			configProvider:     "etcd",
			configPath:         "config/hugo.json",
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
			},
			refreshIterations: 1,
			expectedConfig:    func() *Config { return nil },
			expectedErr:       errors.New("empty endpoint for provider etcd"),
			expectedLogs:      []string{},
		},
		{
			tcase:              "read remote Config + 2 refresh iterations (no changes + error)",
			configBytes:        jsonConfigBytes,
			configer:           configerMock,
			supportedProviders: SupportedProviders,
			supportedExts:      []string{"json"},
			configProvider:     "consul",
			configPath:         "http://127.0.0.1:4001/v1/kv/common/webhooker.json",
			refreshIterations:  2,
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
				c.EXPECT().AddRemoteProvider("consul", "127.0.0.1:4001", "common/webhooker.json").Return(nil)
				c.EXPECT().ReadRemoteConfig().Return(nil)

				v := viper.New()
				v.SetConfigType("json")
				assert.NoError(t, v.ReadConfig(bytes.NewReader(configBytes)))
				conf := &Config{}
				assert.NoError(t, v.Unmarshal(conf))

				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *conf).Return(nil)

				eShell.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
				}).Return(nil).Times(2)
				eJenkins.EXPECT().ValidateParameters(
					map[string]interface{}{
						"job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
						"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
						"endpoint": "https://j.company.com/",
						"login":    "admin",
						"password": "qwerty123",
					}).Return(nil).Times(2)

				// 1st refresh
				c.EXPECT().WatchRemoteConfig().Return(nil)
				newConf := &Config{}
				assert.NoError(t, v.Unmarshal(newConf))
				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *newConf).Return(nil)

				// 2nd refresh
				c.EXPECT().WatchRemoteConfig().Return(errors.New("watch remote Config error"))
			},
			expectedConfig: func() *Config { return getExpectedConfigCompiled(taskExecutors) },
			expectedErr:    nil,
			expectedLogs: []string{
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":1,"level":"info","msg":"starts refreshing Config","params":{"configPath":"common/webhooker.json","configProvider":"consul"}}`,
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":1,"level":"info","msg":"successfully done refreshing Config: no changes","params":{"configPath":"common/webhooker.json","configProvider":"consul"}}`,
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":2,"level":"info","msg":"starts refreshing Config","params":{"configPath":"common/webhooker.json","configProvider":"consul"}}`,
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":2,"level":"error","msg":"Config refresh error: watch remote Config error","params":{"configPath":"common/webhooker.json","configProvider":"consul"}}`,
			},
		},
		{
			tcase:              "read remote Config + 1 refresh iteration with changes",
			configBytes:        jsonConfigBytes,
			configer:           configerMock,
			supportedProviders: SupportedProviders,
			supportedExts:      []string{"json"},
			configProvider:     "consul",
			configPath:         "http://127.0.0.1:4001/v1/kv/common/webhooker.json",
			refreshIterations:  1,
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
				c.EXPECT().AddRemoteProvider("consul", "127.0.0.1:4001", "common/webhooker.json").Return(nil)
				c.EXPECT().ReadRemoteConfig().Return(nil)

				v := viper.New()
				v.SetConfigType("json")
				assert.NoError(t, v.ReadConfig(bytes.NewReader(configBytes)))
				conf := &Config{}
				assert.NoError(t, v.Unmarshal(conf))

				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *conf).Return(nil)

				eShell.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
				}).Return(nil)
				eJenkins.EXPECT().ValidateParameters(
					map[string]interface{}{
						"job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
						"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
						"endpoint": "https://j.company.com/",
						"login":    "admin",
						"password": "qwerty123",
					}).Return(nil)

				// 1st refresh
				c.EXPECT().WatchRemoteConfig().Return(nil)
				newConf := &Config{}
				assert.NoError(t, v.Unmarshal(newConf))
				newConf.Rules = model.Rules{getTestRuleUncompiled(1)}
				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *newConf).Return(nil)

				eShell.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(nil)
			},
			expectedConfig: func() *Config {
				config := getExpectedConfigCompiled(taskExecutors)
				config.Rules = model.Rules{getTestRuleCompiled(1, taskExecutors)}
				return config
			},
			expectedErr: nil,
			expectedLogs: []string{
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":1,"level":"info","msg":"starts refreshing Config","params":{"configPath":"common/webhooker.json","configProvider":"consul"}}`,
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"testrule1","Conditions":{"AlertStatus":"firing","AlertLabels":{"a":"b"},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"aa":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}"},"Block":10000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":1,"level":"info","msg":"successfully done refreshing Config: Config changed","params":{"configPath":"common/webhooker.json","configProvider":"consul"}}`,
			},
		},
		{
			tcase:              "add remote provider error",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: SupportedProviders,
			supportedExts:      []string{"json"},
			configProvider:     "consul",
			configPath:         "http://127.0.0.1:4001/v1/kv/common/webhooker.json",
			refreshIterations:  1,
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
				c.EXPECT().AddRemoteProvider("consul", "127.0.0.1:4001", "common/webhooker.json").Return(errors.New("add remote provider error"))
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("add remote provider error"),
			expectedLogs:   []string{},
		},
		{
			tcase:              "read remote Config error",
			configBytes:        []byte("some raw cfg"),
			configer:           configerMock,
			supportedProviders: SupportedProviders,
			supportedExts:      []string{"json"},
			configProvider:     "consul",
			configPath:         "http://127.0.0.1:4001/v1/kv/common/webhooker.json",
			refreshIterations:  1,
			expectFunc: func(c *Mockconfiger, eShell *executor.MockTaskExecutor, eJenkins *executor.MockTaskExecutor, configBytes []byte, t *testing.T) {
				c.EXPECT().SetConfigType("json")
				c.EXPECT().AddRemoteProvider("consul", "127.0.0.1:4001", "common/webhooker.json").Return(nil)
				c.EXPECT().ReadRemoteConfig().Return(errors.New("read remote Config error"))
			},
			expectedConfig: func() *Config { return nil },
			expectedErr:    errors.New("read remote Config error"),
			expectedLogs:   []string{},
		},
	}

	for _, testUnit := range testTable {
		if testUnit.configProvider != ProviderFile && testUnit.refreshIterations == 0 {
			t.Fatal("test remote provider with zero refresh iterations is not supported")
		}

		logger, hook := test.NewNullLogger()
		logger.Formatter = &logrus.JSONFormatter{DisableTimestamp: true}

		testUnit.expectFunc(configerMock, executorShellMock, executorJenkinsMock, testUnit.configBytes, t)
		readFileFunc := func(filename string) ([]byte, error) {
			if testUnit.readFileFuncErr != nil {
				return nil, testUnit.readFileFuncErr
			}
			_, path, _, err := utils.ParsePath(testUnit.configPath, defaultConfigType, testUnit.configProvider)
			assert.NoError(t, err)
			if filename == path {
				return testUnit.configBytes, nil
			}
			return nil, errors.New("readFileFunc error")
		}

		config, err := New(
			readFileFunc,
			testUnit.configer,
			testUnit.supportedProviders,
			testUnit.supportedExts,
			testUnit.configProvider,
			testUnit.configPath,
			logger,
			taskExecutors,
			testUnit.refreshIterations,
		)

		for i := 0; i < testUnit.refreshIterations; i++ {
			time.Sleep(1 * time.Millisecond) // testing refresh daemon
		}

		assert.Equal(t, testUnit.expectedConfig(), config, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
		assert.Equal(t, expectedLogsFix(testUnit.expectedLogs), logsFromHook(t, hook), testUnit.tcase)
	}
}

func TestRefreshDaemon(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configerMock := NewMockconfiger(ctrl)

	executorMock := executor.NewMockTaskExecutor(ctrl)

	taskExecutors := map[string]executor.TaskExecutor{
		"shell": executorMock,
	}

	type testTableData struct {
		tcase             string
		provider, path    string
		refreshIterations int
		expectFunc        func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config)
		newConfig         func() *Config
		expectedRules     model.Rules
		expectedLogs      []string
	}

	testTable := []testTableData{
		{
			tcase:             "refresh with change",
			provider:          "consul",
			path:              "https://consul/test.json",
			refreshIterations: 1,
			expectFunc: func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config) {
				c.EXPECT().WatchRemoteConfig().Return(nil)
				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *ec).Return(nil)
				e.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(nil)
			},
			newConfig: func() *Config {
				config := getExpectedConfigUncompiled()
				config.Rules = model.Rules{getTestRuleUncompiled(1)}
				return config
			},
			expectedRules: model.Rules{getTestRuleCompiled(1, taskExecutors)},
			expectedLogs: []string{
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":null}]}]},"context":"startup","iteration":1,"level":"info","msg":"starts refreshing Config","params":{"configPath":"https://consul/test.json","configProvider":"consul"}}`,
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"testrule1","Conditions":{"AlertStatus":"firing","AlertLabels":{"a":"b"},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"aa":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}"},"Block":10000000000,"TaskExecutor":{}}]}]},"context":"startup","iteration":1,"level":"info","msg":"successfully done refreshing Config: Config changed","params":{"configPath":"https://consul/test.json","configProvider":"consul"}}`,
			},
		},
		{
			tcase:             "error",
			provider:          "consul",
			path:              "https://consul/test.json",
			refreshIterations: 1,
			expectFunc: func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config) {
				c.EXPECT().WatchRemoteConfig().Return(errors.New("error"))
			},
			newConfig:     func() *Config { return nil },
			expectedRules: getExpectedConfigCompiled(taskExecutors).Rules,
			expectedLogs: []string{
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":null}]}]},"context":"startup","iteration":1,"level":"info","msg":"starts refreshing Config","params":{"configPath":"https://consul/test.json","configProvider":"consul"}}`,
				`{"Config":{"BlockCacheSize":104857600,"PoolSize":100,"Runners":30,"RemoteConfigRefreshInterval":1,"CommonParameters":{"jenkins1":{"endpoint":"https://j.company.com/","login":"admin","password":"qwerty123"}},"Rules":[{"Name":"LowDiskSpaceFix","Conditions":{"AlertStatus":"firing","AlertLabels":{"alertname":"LowDiskSpace"},"AlertLabelsRegexp":{"instance":{}},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_enabled":{}}},"Actions":[{"Executor":"shell","CommonParameters":"","Parameters":{"command":"./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}"},"Block":600000000000,"TaskExecutor":{}}]},{"Name":"AnyAlertFix","Conditions":{"AlertStatus":"firing","AlertLabels":{},"AlertLabelsRegexp":{},"AlertAnnotations":{},"AlertAnnotationsRegexp":{"webhooker_job":{}}},"Actions":[{"Executor":"jenkins","CommonParameters":"jenkins1","Parameters":{"endpoint":"https://j.company.com/","instance":"${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}","job_name":"${ANNOTATIONS_WEBHOOKER_JOB}","login":"admin","password":"qwerty123"},"Block":300000000000,"TaskExecutor":null}]}]},"context":"startup","iteration":1,"level":"error","msg":"Config refresh error: error","params":{"configPath":"https://consul/test.json","configProvider":"consul"}}`,
			},
		},
	}

	for _, testUnit := range testTable {
		if testUnit.refreshIterations == 0 {
			t.Fatal("test with zero refresh iterations is not supported")
		}

		logger, hook := test.NewNullLogger()
		logger.Formatter = &logrus.JSONFormatter{DisableTimestamp: true}

		testUnit.expectFunc(configerMock, executorMock, testUnit.newConfig())
		config := getExpectedConfigCompiled(taskExecutors)
		refreshDaemon(config, testUnit.provider, testUnit.path, configerMock, logger, taskExecutors, testUnit.refreshIterations)

		assert.Equal(t, testUnit.expectedRules, config.Rules, testUnit.tcase)
		assert.Equal(t, expectedLogsFix(testUnit.expectedLogs), logsFromHook(t, hook), testUnit.tcase)
	}
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configerMock := NewMockconfiger(ctrl)

	executorMock := executor.NewMockTaskExecutor(ctrl)

	taskExecutors := map[string]executor.TaskExecutor{
		"shell": executorMock,
	}

	type testTableData struct {
		tcase           string
		config          *Config
		expectFunc      func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config)
		newConfig       func() *Config
		expectedRules   model.Rules
		expectedChanged bool
		expectedErr     error
	}

	testTable := []testTableData{
		{
			tcase:  "refreshed",
			config: getExpectedConfigCompiled(taskExecutors),
			expectFunc: func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config) {
				c.EXPECT().WatchRemoteConfig().Return(nil)
				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *ec).Return(nil)
				e.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(nil)
			},
			newConfig: func() *Config {
				config := getExpectedConfigUncompiled()
				config.Rules = model.Rules{getTestRuleUncompiled(1)}
				return config
			},
			expectedRules:   model.Rules{getTestRuleCompiled(1, taskExecutors)},
			expectedChanged: true,
			expectedErr:     nil,
		},
		{
			tcase:  "watch remote Config error",
			config: getExpectedConfigCompiled(taskExecutors),
			expectFunc: func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config) {
				c.EXPECT().WatchRemoteConfig().Return(errors.New("error"))
			},
			newConfig:       func() *Config { return nil },
			expectedRules:   getExpectedConfigCompiled(taskExecutors).Rules,
			expectedChanged: false,
			expectedErr:     errors.New("error"),
		},
		{
			tcase:  "unmarshall Config error",
			config: getExpectedConfigCompiled(taskExecutors),
			expectFunc: func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config) {
				c.EXPECT().WatchRemoteConfig().Return(nil)
				c.EXPECT().Unmarshal(&Config{}).Return(errors.New("unmarshall error"))
			},
			newConfig:       func() *Config { return nil },
			expectedRules:   getExpectedConfigCompiled(taskExecutors).Rules,
			expectedChanged: false,
			expectedErr:     errors.New("unmarshall error"),
		},
		{
			tcase:  "prepare rules error",
			config: getExpectedConfigCompiled(taskExecutors),
			expectFunc: func(c *Mockconfiger, e *executor.MockTaskExecutor, ec *Config) {
				c.EXPECT().WatchRemoteConfig().Return(nil)
				c.EXPECT().Unmarshal(&Config{}).SetArg(0, *ec).Return(nil)
			},
			newConfig: func() *Config {
				config := getExpectedConfigUncompiled()
				config.Rules = nil
				return config
			},
			expectedRules:   getExpectedConfigCompiled(taskExecutors).Rules,
			expectedChanged: false,
			expectedErr:     errors.New("empty rules list"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(configerMock, executorMock, testUnit.newConfig())
		changed, err := refresh(testUnit.config, configerMock, taskExecutors)
		assert.Equal(t, testUnit.expectedChanged, changed, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
		assert.Equal(t, testUnit.expectedRules, testUnit.config.Rules, testUnit.tcase)
	}
}

func TestConfig_fillDefaults(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		config   Config
		expected Config
	}

	testTable := []testTableData{
		{
			tcase: "fill BlockCacheSize",
			config: Config{
				Runners: 10,
			},
			expected: Config{
				BlockCacheSize: defaultBlockCacheSize,
				Runners:        10,
			},
		},
		{
			tcase: "fill Runners",
			config: Config{
				BlockCacheSize: 10 * 1024 * 1024,
			},
			expected: Config{
				BlockCacheSize: 10 * 1024 * 1024,
				Runners:        defaultRunners,
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.config.fillDefaults()
		assert.Equal(t, testUnit.expected, testUnit.config, testUnit.tcase)
	}
}

func logsFromHook(t *testing.T, hook *test.Hook) (logs []string) {
	if hook == nil {
		return []string{}
	}

	if hook.Entries == nil {
		return []string{}
	}

	logs = make([]string, len(hook.Entries))
	for i, entry := range hook.Entries {
		log, err := entry.String()
		assert.Equal(t, nil, err)
		logs[i] = log
	}
	return
}

func expectedLogsFix(logs []string) (expectedLogs []string) {
	expectedLogs = make([]string, len(logs))
	for i, log := range logs {
		expectedLogs[i] = log + "\n"
	}
	return
}

func getExpectedConfigCompiled(taskExecutors map[string]executor.TaskExecutor) *Config {
	config := getExpectedConfigUncompiled()
	config.Rules = []model.Rule{
		{
			Name: "LowDiskSpaceFix",
			Conditions: model.Conditions{
				AlertStatus: "firing",
				AlertLabels: map[string]string{
					"alertname": "LowDiskSpace",
				},
				AlertLabelsRegexp: map[string]*regexp.Regexp{
					"instance": regexp.MustCompile("^logs_(.*?)"),
				},
				AlertAnnotations: map[string]string{},
				AlertAnnotationsRegexp: map[string]*regexp.Regexp{
					"webhooker_enabled": regexp.MustCompile("(.*?)")},
			},
			Actions: model.Actions{
				{
					Executor: "shell",
					Parameters: map[string]interface{}{
						"command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
					},
					Block:        10 * time.Minute,
					TaskExecutor: taskExecutors["shell"],
				},
			},
		},
		{
			Name: "AnyAlertFix",
			Conditions: model.Conditions{
				AlertStatus:       "firing",
				AlertLabels:       map[string]string{},
				AlertLabelsRegexp: map[string]*regexp.Regexp{},
				AlertAnnotations:  map[string]string{},
				AlertAnnotationsRegexp: map[string]*regexp.Regexp{
					"webhooker_job": regexp.MustCompile("(.*?)"),
				},
			},
			Actions: model.Actions{
				{
					Executor:         "jenkins",
					CommonParameters: "jenkins1",
					Parameters: map[string]interface{}{
						"job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
						"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
						"endpoint": "https://j.company.com/",
						"login":    "admin",
						"password": "qwerty123",
					},
					Block:        5 * time.Minute,
					TaskExecutor: taskExecutors["jenkins"],
				},
			},
		},
	}
	return config
}

func getExpectedConfigUncompiled() *Config {
	return &Config{
		BlockCacheSize:              104857600,
		PoolSize:                    100,
		Runners:                     30,
		RemoteConfigRefreshInterval: 1 * time.Nanosecond,
		CommonParameters: map[string]map[string]interface{}{
			"jenkins1": {
				"endpoint": "https://j.company.com/",
				"login":    "admin",
				"password": "qwerty123",
			},
		},
		Rules: []model.Rule{
			{
				Name: "LowDiskSpaceFix",
				Conditions: model.Conditions{
					AlertStatus: "firing",
					AlertLabels: map[string]string{
						"alertname": "LowDiskSpace",
						"instance":  "^logs_(.*?)",
					},
					AlertLabelsRegexp: map[string]*regexp.Regexp{},
					AlertAnnotations: map[string]string{
						"webhooker_enabled": "(.*?)",
					},
					AlertAnnotationsRegexp: map[string]*regexp.Regexp{},
				},
				Actions: model.Actions{
					{
						Executor: "shell",
						Parameters: map[string]interface{}{
							"command": "./clean_server.sh ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
						},
						Block: 10 * time.Minute,
					},
				},
			},
			{
				Name: "AnyAlertFix",
				Conditions: model.Conditions{
					AlertStatus:       "firing",
					AlertLabels:       map[string]string{},
					AlertLabelsRegexp: map[string]*regexp.Regexp{},
					AlertAnnotations: map[string]string{
						"webhooker_command": "(.*?)",
					},
					AlertAnnotationsRegexp: map[string]*regexp.Regexp{},
				},
				Actions: model.Actions{
					{
						Executor:         "jenkins",
						CommonParameters: "jenkins1",
						Parameters: map[string]interface{}{
							"job_name": "${ANNOTATIONS_WEBHOOKER_JOB}",
							"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
							"endpoint": "https://j.company.com/",
							"login":    "admin",
							"password": "qwerty123",
						},
						Block: 5 * time.Minute,
					},
				},
			},
		},
	}
}

func getTestRuleUncompiled(num int) model.Rule {
	return model.Rule{
		Name: fmt.Sprintf("testrule%v", num),
		Conditions: model.Conditions{
			AlertStatus: "firing",
			AlertLabels: map[string]string{
				"a": "b",
			},
			AlertLabelsRegexp: map[string]*regexp.Regexp{},
			AlertAnnotations: map[string]string{
				"aa": "ab(.*?)",
			},
			AlertAnnotationsRegexp: map[string]*regexp.Regexp{},
		},
		Actions: model.Actions{
			{
				Executor: "shell",
				Parameters: map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				},
				Block: 10 * time.Second,
			},
		},
	}
}

func getTestRuleCompiled(num int, taskExecutors map[string]executor.TaskExecutor) model.Rule {
	return model.Rule{
		Name: fmt.Sprintf("testrule%v", num),
		Conditions: model.Conditions{
			AlertStatus: "firing",
			AlertLabels: map[string]string{
				"a": "b",
			},
			AlertLabelsRegexp: map[string]*regexp.Regexp{},
			AlertAnnotations:  map[string]string{},
			AlertAnnotationsRegexp: map[string]*regexp.Regexp{
				"aa": regexp.MustCompile("ab(.*?)"),
			},
		},
		Actions: model.Actions{
			{
				Executor: "shell",
				Parameters: map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				},
				Block:        10 * time.Second,
				TaskExecutor: taskExecutors["shell"],
			},
		},
	}
}

// Test mock for coverage
func TestMockConfigerCoverage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configerMock := NewMockconfiger(ctrl)
	configerMock.EXPECT().Unmarshal(nil, nil)
	configerMock.Unmarshal(nil, nil)
}
