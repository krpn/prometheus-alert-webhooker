package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/coocood/freecache"
	blc "github.com/lohmag/prometheus-alert-webhooker/blocker"
	cfg "github.com/lohmag/prometheus-alert-webhooker/config"
	"github.com/lohmag/prometheus-alert-webhooker/executor"
	"github.com/lohmag/prometheus-alert-webhooker/executor/http"
	"github.com/lohmag/prometheus-alert-webhooker/executor/jenkins"
	"github.com/lohmag/prometheus-alert-webhooker/executor/shell"
	"github.com/lohmag/prometheus-alert-webhooker/executor/telegram"
	mtrc "github.com/lohmag/prometheus-alert-webhooker/metric"
	"github.com/lohmag/prometheus-alert-webhooker/model"
	"github.com/lohmag/prometheus-alert-webhooker/runner"
	"github.com/lohmag/prometheus-alert-webhooker/webhook"
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
		"http": httpe.NewExecutor(func(timeout time.Duration) httpe.Doer {
			return &http.Client{Timeout: timeout}
		}),
	}
)

const context = "startup"
const realRun = 0

func main() {
	_ = kingpin.Parse()

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	ctxLogger := logger.WithFields(logrus.Fields{
		"context": context,
		"params": map[string]interface{}{
			"listenAddr":     listenAddr,
			"configProvider": configProvider,
			"configPath":     configPath,
		},
	})

	ctxLogger.Debug("starting up service, prepare config")

	config, err := cfg.New(
		ioutil.ReadFile,
		viper.New(),
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
	ctxLogger.Debug("config prepared")

	var (
		tasksCh = make(chan model.Tasks, config.PoolSize)
		blocker = blc.New(freecache.NewCache(config.BlockCacheSize))
		metric  = mtrc.New()
	)

	// runner
	ctxLogger.Debug("starting up runners")
	go runner.Start(config.Runners, tasksCh, blocker, metric, logger, time.Now)

	// HTTP
	ctxLogger.Debug("starting up wehbook")
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/webhooker", func(_ http.ResponseWriter, r *http.Request) {
		webhook.Webhook(r, config.Rules, tasksCh, metric, logger, time.Now)
	})
	ctxLogger.Fatalf("http server startup error: %v", http.ListenAndServe(*listenAddr, nil))
}
