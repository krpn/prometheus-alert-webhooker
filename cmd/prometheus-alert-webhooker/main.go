package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/coocood/freecache"
	blc "github.com/krpn/prometheus-alert-webhooker/blocker"
	cfg "github.com/krpn/prometheus-alert-webhooker/config"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/executor/jenkins"
	"github.com/krpn/prometheus-alert-webhooker/executor/shell"
	"github.com/krpn/prometheus-alert-webhooker/executor/telegram"
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
	"os/exec"
	"time"
)

var (
	listenAddr     = kingpin.Flag("listen", "HTTP port to listen on").Default(":8080").Short('l').String()
	configProvider = kingpin.Flag("provider", "Config provider: file, etcd, consul").Default(cfg.ProviderFile).Short('p').String()
	configPath     = kingpin.Flag("config", "Path to config file with extension, can be link for etcd, consul providers").Default("config/config.yaml").Short('c').String()
	verbose        = kingpin.Flag("verbose", "Enable verbose logging").Default("false").Short('v').Bool()

	taskExecutors = map[string]executor.TaskExecutor{
		"shell":    shell.NewExecutor(exec.Command),
		"jenkins":  jenkins.NewExecutor(),
		"telegram": telegram.NewExecutor(&http.Client{}),
	}
)

const context = "startup"
const realRun = 0

func main() {
	_ = kingpin.Parse()

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
