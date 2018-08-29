package webhook

import (
	"encoding/json"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const context = "webhook"

// Webhook is a handler for Alertmanager payload.
func Webhook(req *http.Request, rules model.Rules, tasksCh chan executor.Task, metric metricser, logger *logrus.Logger, nowFunc func() time.Time) {
	decoder := json.NewDecoder(req.Body)

	payload := &model.Payload{}
	if err := decoder.Decode(payload); err != nil {
		return
	}

	eventID := getEventID(nowFunc)

	alerts := payload.ToAlerts()
	tasks := alerts.ToTasks(rules, eventID)

	ctxLogger := logger.WithField("context", context)
	payloadLogger := ctxLogger.WithFields(
		logrus.Fields{
			"event_id": eventID,
			"payload":  payload,
			"tasks":    tasks.Details(),
		},
	)
	if len(tasks) == 0 {
		payloadLogger.Info("payload is received, no tasks for it")
		return
	}

	payloadLogger.Info("payload is received, tasks are prepared")

	for _, task := range tasks {
		taskLogger := ctxLogger.WithFields(executor.TaskDetails(task))
		taskLogger.Info("ready to send task to runner")

		tasksCh <- task

		taskLogger.Info("sent task to runner")
		metric.IncomeTaskInc(task.Rule(), task.Alert(), task.ExecutorName())
	}

	payloadLogger.Info("all tasks sent to runners")
}

func getEventID(nowFunc func() time.Time) string {
	return utils.MD5HashFromTime(nowFunc())[0:4]
}

//go:generate mockgen -source=webhook.go -destination=webhook_mocks.go -package=webhook doc github.com/golang/mock/gomock

type metricser interface {
	IncomeTaskInc(rule, alert, executor string)
}
