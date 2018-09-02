package main

import (
	"flag"
	"fmt"
	"github.com/coocood/freecache"
	blc "github.com/krpn/prometheus-alert-webhooker/blocker"
	cfg "github.com/krpn/prometheus-alert-webhooker/config"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/executor/jenkins"
	"github.com/krpn/prometheus-alert-webhooker/executor/shell"
	mtrc "github.com/krpn/prometheus-alert-webhooker/metric"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/krpn/prometheus-alert-webhooker/runner"
	"github.com/krpn/prometheus-alert-webhooker/webhook"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var (
	listenAddr     = flag.String("l", ":8080", "HTTP port to listen on")
	configProvider = flag.String("p", cfg.ProviderFile, "Config provider: file, etcd, consul")
	configPath     = flag.String("c", "config/config.yaml", "Path to config file with extension, can be link for etcd, consul providers")
	verbose        = flag.Bool("v", false, "Enable verbose logging")

	taskExecutors = map[string]executor.TaskExecutor{
		"shell":   shell.NewExecutor(exec.Command),
		"jenkins": jenkins.NewExecutor(),
	}
)

const context = "startup"
const realRun = 0

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	if *verbose {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(logrus.WarnLevel)
	}

	ctxLogger := logger.WithFields(logrus.Fields{
		"context": context,
		"params": map[string]interface{}{
			"listenAddr":     listenAddr,
			"configProvider": configProvider,
			"configPath":     configPath,
		},
	})

	ctxLogger.Info("starting up service, prepare config")

	config, err := cfg.New(
		ioutil.ReadFile,
		viper.New(),
		cfg.SupportedProviders,
		viper.SupportedExts,
		*configProvider,
		*configPath,
		logger,
		taskExecutors,
		realRun,
	)
	if err != nil {
		ctxLogger.Fatalf("create config error: %v", err)
	}

	ctxLogger = ctxLogger.WithField("config", config)
	ctxLogger.Info("config prepared")

	var (
		tasksCh = make(chan model.Tasks, config.PoolSize)
		blocker = blc.New(freecache.NewCache(config.BlockCacheSize))
		metric  = mtrc.New()
	)

	// runner
	ctxLogger.Info("starting up runners")
	go runner.Start(config.Runners, tasksCh, blocker, metric, logger, time.Now)

	// HTTP
	ctxLogger.Info("starting up wehbook")
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/webhooker", func(w http.ResponseWriter, r *http.Request) {
		webhook.Webhook(r, config.Rules, tasksCh, metric, logger, time.Now)
	})
	ctxLogger.Fatalf("http server startup error: %v", http.ListenAndServe(*listenAddr, nil))
}
