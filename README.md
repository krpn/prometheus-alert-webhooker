# prometheus-alert-webhooker

[![License](https://img.shields.io/dub/l/vibe-d.svg)](https://github.com/krpn/prometheus-alert-webhooker/blob/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/krpn/prometheus-alert-webhooker)](https://goreportcard.com/report/github.com/krpn/prometheus-alert-webhooker)

Convert [Prometheus Alertmanager Webhook](https://prometheus.io/docs/operating/integrations/#alertmanager-webhook-receiver) to any action

# Features

* Converts Prometheus Alertmanager Webhook to any action using rules
* Currently supports action types:
    * run Jenkins job (with parameters)
    * run shell command
* Alert labels/annotations can be used in action placeholders
* Rules are set in config and can be flex ([example](https://github.com/krpn/prometheus-alert-webhooker/blob/master/example/config.yaml))
* Supported config types JSON, TOML, YAML, HCL, and Java properties ([Viper](https://github.com/spf13/viper) is used)
* Supported config providers: file, etcd, consul (with automatic refresh)
* Prometheus metrics built in

# Quick Start