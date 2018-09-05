package webhook

import (
	"encoding/json"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const context = "webhook"

// Webhook is a handler for Alertmanager payload.
func Webhook(req *http.Request, rules model.Rules, tasksCh chan model.Tasks, metric metricser, logger *logrus.Logger, nowFunc func() time.Time) {
	decoder := json.NewDecoder(req.Body)

	payload := &model.Payload{}
	if err := decoder.Decode(payload); err != nil {
		return
	}

	eventID := getEventID(nowFunc)

	alerts := payload.ToAlerts()
	tasksGroups := alerts.ToTasksGroups(rules, eventID)

	ctxLogger := logger.WithField("context", context)
	payloadLogger := ctxLogger.WithFields(
		logrus.Fields{
			"event_id":     eventID,
			"payload":      payload,
			"tasks_groups": tasksGroups.Details(),
		},
	)
	if len(tasksGroups) == 0 {
		payloadLogger.Debug("payload is received, no tasks for it")
		return
	}

	payloadLogger.Debug("payload is received, tasks are prepared")

	for _, tasks := range tasksGroups {
		tasksLogger := ctxLogger.WithField("tasks", tasks.Details())
		tasksLogger.Debug("ready to send tasks to runner")

		tasksCh <- tasks

		tasksLogger.Debug("sent tasks to runner")

		for _, task := range tasks {
			metric.IncomeTaskInc(task.Rule(), task.Alert(), task.ExecutorName())
		}
	}

	payloadLogger.Debug("all tasks sent to runners")
}

func getEventID(nowFunc func() time.Time) string {
	return utils.MD5HashFromTime(nowFunc())[0:4]
}

//go:generate mockgen -source=webhook.go -destination=webhook_mocks.go -package=webhook doc github.com/golang/mock/gomock

type metricser interface {
	IncomeTaskInc(rule, alert, executor string)
}
