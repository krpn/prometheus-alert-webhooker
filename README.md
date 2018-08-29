# prometheus-alert-webhooker

[![Build Status](https://travis-ci.org/krpn/prometheus-alert-webhooker.svg?branch=master)](https://travis-ci.org/krpn/prometheus-alert-webhooker)
[![Go Report Card](https://goreportcard.com/badge/github.com/krpn/prometheus-alert-webhooker)](https://goreportcard.com/report/github.com/krpn/prometheus-alert-webhooker)
[![Coverage Status](https://coveralls.io/repos/github/krpn/prometheus-alert-webhooker/badge.svg?branch=master)](https://coveralls.io/github/krpn/prometheus-alert-webhooker?branch=master)
[![Docker Image](https://images.microbadger.com/badges/image/krpn/prometheus-alert-webhooker.svg)](https://microbadger.com/images/krpn/prometheus-alert-webhooker)
[![License](https://img.shields.io/github/license/krpn/prometheus-alert-webhooker.svg)](https://github.com/krpn/prometheus-alert-webhooker/blob/master/LICENSE)

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
* A docker image available on [Docker Hub](https://hub.docker.com/r/krpn/prometheus-alert-webhooker/)

# Quick Start

1. Prepare config.yaml file based on [example](https://github.com/krpn/prometheus-alert-webhooker/blob/master/example/config.yaml)

2. Run container with command:

    `docker run -d -p <port>:8080 -v <path to config.yaml>:/config --name prometheus-alert-webhooker krpn/prometheus-alert-webhooker -v`

3. Check logs:

    `docker logs prometheus-alert-webhooker`

4. Add webhook to [Alertmanager webhook config](https://prometheus.io/docs/alerting/configuration/#webhook_config). url will be:

    `url: http://<server container runned on>:<port>/webhooker`
    
5. Add webhooker instance to [Prometheus scrape targets](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#%3Cscrape_config%3E) if needed (port is the same)