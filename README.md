# prometheus-alert-webhooker

[![Build Status](https://travis-ci.org/krpn/prometheus-alert-webhooker.svg?branch=master)](https://travis-ci.org/krpn/prometheus-alert-webhooker) [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=krpn_prometheus-alert-webhooker&metric=alert_status)](https://sonarcloud.io/dashboard?id=krpn_prometheus-alert-webhooker) [![Coverage Status](https://sonarcloud.io/api/project_badges/measure?project=krpn_prometheus-alert-webhooker&metric=coverage)](https://sonarcloud.io/component_measures?id=krpn_prometheus-alert-webhooker&metric=coverage) [![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=krpn_prometheus-alert-webhooker&metric=sqale_index)](https://sonarcloud.io/component_measures?id=krpn_prometheus-alert-webhooker&metric=sqale_index) [![License](https://img.shields.io/github/license/krpn/prometheus-alert-webhooker.svg)](https://github.com/krpn/prometheus-alert-webhooker/blob/master/LICENSE)

prometheus-alert-webhooker converts [Prometheus Alertmanager Webhook](https://prometheus.io/docs/operating/integrations/#alertmanager-webhook-receiver) to any action

# Table of Contents
* [Features](#features)
* [Quick Start](#quick-start)
* [Configuration](#configuration)
* [Executors](#executors)
    * [Executor `jenkins`](#executor-jenkins)
    * [Executor `shell`](#executor-shell)
    * [Executor `http`](#executor-http)
    * [Executor `telegram`](#executor-telegram)
* [Command-Line Flags](#command-line-flags)
* [Exposed Prometheus Metrics](#exposed-prometheus-metrics)
* [Contribute](#contribute)

# Features

* Converts Prometheus Alertmanager Webhook to any action using rules
* Currently supports actions (see [executors](#executors)):
    * run Jenkins job (optionally with parameters)
    * run shell command
    * send Telegram message
* Alert labels/annotations can be used in action placeholders
* Rules are set in config and can be flexible ([example](https://github.com/krpn/prometheus-alert-webhooker/blob/master/example/config.yaml))
* Supported config types JSON, TOML, YAML, HCL, and Java properties ([Viper](https://github.com/spf13/viper) is used)
* Supported config providers: file, etcd, consul (with automatic refresh)
* Prometheus metrics built in
* A docker image available on [Docker Hub](https://hub.docker.com/r/krpn/prometheus-alert-webhooker/)

[(back to top)](#prometheus-alert-webhooker)

# Quick Start

1. Prepare config.yaml file based on [example](https://github.com/krpn/prometheus-alert-webhooker/blob/master/example/config.yaml) (details in [configuration](#configuration))

2. Run container with command ([cli flags](#command-line-flags)):

    If you use file config:
    
    `docker run -d -p <port>:8080 -v <path to config.yaml>:/config --name prometheus-alert-webhooker krpn/prometheus-alert-webhooker --verbose`
    
    If you use Consul:
    
    `docker run -d -p <port>:8080 --name prometheus-alert-webhooker krpn/prometheus-alert-webhooker --verbose --provider consul --config http://<consul address>:8500/v1/kv/<path to config>`

3. Checkout logs:

    `docker logs prometheus-alert-webhooker`

4. Add webhook to [Alertmanager webhook config](https://prometheus.io/docs/alerting/configuration/#webhook_config). url will be:

    `url: http://<server container runned on>:<port>/webhooker`
    
5. Add webhooker instance to [Prometheus scrape targets](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#%3Cscrape_config%3E) if needed (port is the same; [metrics](#exposed-prometheus-metrics))

[(back to top)](#prometheus-alert-webhooker)

# Configuration

Configuration description based on YAML format:

```yaml
# WEBHOOKER GLOBAL SETTINGS
# cache size for blocked tasks
# calculate: 50 * 1024 * 1024 = 50 MB
# default if not set: 52428800
block_cache_size: 52428800

# pool size for new tasks
# locks webhook if overflow
# default if not set: 100
pool_size: 100

# runners count for parallel actions execute
# default if not set: 10
runners: 10

# remote config refresh interval
# used only for etcd and consul config providers
# rules including common parameters will be refreshed only
# global settings exclude refresh interval will NOT be refreshed (restart is required)
# will not refresh if zero
# default if not set: 0s
remote_config_refresh_interval: 60s


# COMMON PARAMETERS FOR ACTIONS (optional)
# available to get in rules-actions-common_parameters
# can be used for storing credentials
common_parameters:
  # name of parameters set
  <parameters_set_1>:
    <parameter_1>: <parameter_1_value>
    <parameter_n>: <parameter_n_value>


# LIST OF RULES
rules:
- name: <rule_1> # rule name

  # list of conditions for this rule
  # values can be regexp
  # regexp detecting by existence of regexp group
  # if no regexp groups found value used as string
  # matching with AND operator, ALL conditions must match
  conditions:
    # define alert status for match if needed
    # default if not set: firing
    # alert_status: firing
    
    # list of alert labels for match
    alert_labels:
      <label_1>: <label_value_1>
      <label_n>: <label_value_n>
      
    # list of alert annotations for match
    alert_annotations:
      <annotation_1>: <annotation_value_1>
      <annotation_n>: <annotation_value_n>
  
  # list of actions for this rule
  # (!) if few actions are match for alert all matched actions will be exec
  # (!) actions will be execute sequentially
  # if action fails the other actions will be cancelled
  actions:
  - executor: <executor> # executor from available executor list 
    
    # get parameters from common if needed
    # common parameters has low priority to action parameters:
    #   the same parameter will be replaced by action parameter
    # common_parameters: <parameters_set_1>
    
    # list of parameters for action
    # (!) each executor can have a list of required parameters
    # parameter values can contains placeholders fully in UPPER case:
    #   ${LABELS_<LABEL_N>} will be replaced by <label_value_n>
    #   ${ANNOTATIONS_<ANNOTATION_N>} will be replaced by <annotation_value_n>
    # each placeholder can have one modificator (optionally): ${<MODIFICATOR>LABELS_<LABEL_N>}
    # <MODIFICATOR> list:
    #   URLENCODE_            - escapes the string so it can be safely placed inside a URL query
    #   CUT_AFTER_LAST_COLON_ - cuts text after last colon, can be used for cut port from instance label
    #   JSON_ESCAPE_          - escapes the string so it can be safely placed inside a JSON value
    # examples:
    #   ${LABEL_ALERTNAME} - alert name
    #   ${ANNOTATIONS_COMMAND} - value from annotation "command"
    #   ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} - instance without port
    #   ${URLENCODE_ANNOTATIONS_SUMMARY} - urlencoded value from annotation "summary"
    #   ${JSON_ESCAPE_ANNOTATIONS_DESCRIPTION} - JSON escaped value from annotation "description"
    # (!) all unexpected parameters will be ignored
    parameters:
      <parameter_1>: <parameter_1_value>
      <parameter_n>: <parameter_n_value>
    
    # block time for successfully executed action
    # used for occasional exec
    # (!) blocks only unique set of parameters for this action
    # will not block if zero
    # (!) all blocks released when webhooker restarts
    # default if not set: 0s
    block: 10m
```

[(back to top)](#prometheus-alert-webhooker)

# Executors

Executors and it parameters described below.

## Executor `jenkins`

`jenkins` is used for run Jenkins jobs. Runner starts job, waits job finish and check it was successfull.

| Parameter                        | Type       | Description                                                                                                                  | Example                                                        |
|----------------------------------|:----------:|------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------|
| `endpoint`                       | `string`   | Jenkins address                                                                                                              | `endpoint: https://jenkins.example.com/`                       |
| `login`                          | `string`   | Jenkins login                                                                                                                | `login: webhooker`                                             |
| `password`                       | `string`   | Jenkins password                                                                                                             | `password: qwerty123`                                          |
| `job`                            | `string`   | Name of job to run. If you use Jenkins Folders Plugin you need set the full path to job                                      | `job: YourJob or Folder/job/YourJob (Folders Plugin)`          |
| `job parameter <parameter_name>` | `string`   | (optional) Pass <parameter_name> to job                                                                                      | `job parameter server: ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}` |
| `state_refresh_delay`            | `duration` | (optional, default: 15s) How often runner will be refresh job status when executing                                          | `state_refresh_delay: 3s`                                      |
| `secure_interations_limit`       | `integer`  | (optional, default: 1000) How many refresh status iterations will be until Job will be considered hung and runner release it | `secure_interations_limit: 500`                                |

## Executor `shell`

`shell` is used for run unix shell command. *Remember: all shell scripts must be mounted if you use Docker.*

| Parameter | Type     | Description         | Example                               |
|-----------|:--------:|---------------------|---------------------------------------|
| `command` | `string` | Command for execute | `command: ./clean.sh ${LABEL_FOLDER}` |

## Executor `http`

`http` is used for making HTTP requests.

| Parameter              | Type       | Description                                                                                  | Example                                                    |
|------------------------|:----------:|----------------------------------------------------------------------------------------------|------------------------------------------------------------|
| `url`                  | `string`   | Request URL                                                                                  | `url: https://www.example.com/`                            |
| `method`               | `string`   | (optional, default: GET) Request method                                                      | `method: POST`                                             |
| `body`                 | `string`   | (optional) Request body                                                                      | `body: {"data": "${JSON_ESCAPE_ANNOTATIONS_DESCRIPTION}"}` |
| `header <header_name>` | `string`   | (optional) Sets header <header_name>                                                         | `header Authorization: ba0828c9fac6b0b47d9147963429d091`   |
| `timeout`              | `duration` | (optional, default: 1s) Request timeout                                                      | `timeout: 100ms`                                           |
| `success_http_status`  | `integer`  | (optional, default: 200) Success response status code, will be checked after request execute | `success_http_status: 201`                                 |

## Executor `telegram`

`telegram` is used for handy notifications about webhooker events.

| Parameter   | Type      | Description                                        | Example                                                |
|-------------|:---------:|----------------------------------------------------|--------------------------------------------------------|
| `bot_token` | `string`  | Bot token from [BotFather](https://t.me/BotFather) | `bot_token: 123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11` |
| `chat_id`   | `integer` | Chat ID for send notifications to                  | `chat_id: -1001103941234`                              |
| `message`   | `string`  | Message for send                                   | `message: Fixed ${LABEL_ALERTNAME}`                    |

[(back to top)](#prometheus-alert-webhooker)

# Command-Line Flags

Usage: `prometheus-alert-webhooker [<flags>]`

| Flag                 | Type     | Description                                                                | Default              |
|----------------------|:--------:|----------------------------------------------------------------------------|----------------------|
| `-p` or `--provider` | `string` | Config provider: file, etcd, consul                                        | `file`               |
| `-c` or `--config`   | `string` | Path to config file with extension, can be link for etcd, consul providers | `config/config.yaml` |
| `-l` or `--listen`   | `string` | HTTP port to listen on                                                     | `:8080`              |
| `-v` or `--verbose`  |          | Enable verbose logging                                                     |                      |
| `--help`             |          | Show help                                                                  |                      |

[(back to top)](#prometheus-alert-webhooker)

# Exposed Prometheus Metrics

| Name                                        | Description                                                                                    | Labels                                     |
|---------------------------------------------|------------------------------------------------------------------------------------------------|--------------------------------------------|
| `prometheus_alert_webhooker_income_tasks`   | Income tasks counter                                                                           | `rule` `alert` `executor`                  |
| `prometheus_alert_webhooker_executed_tasks` | Executed tasks histogram with duration in seconds. `error` label is empty if no error occurred | `rule` `alert` `executor` `result` `error` |

[(back to top)](#prometheus-alert-webhooker)

# Contribute

Please feel free to send me [pull requests](https://github.com/krpn/prometheus-alert-webhooker/pulls).

[(back to top)](#prometheus-alert-webhooker)