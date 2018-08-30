package metric

import (
	pr "github.com/prometheus/client_golang/prometheus"
	"time"
)

// PrometheusMetrics describes Prometheus metric collector.
type PrometheusMetrics struct {
	incomeTasks  incomeTasks
	excutedTasks excutedTasks
}

// New creates PrometheusMetrics.
func New() *PrometheusMetrics {
	incomeTasks := pr.NewCounterVec(
		pr.CounterOpts{
			Namespace: "prometheus",
			Subsystem: "alert_webhooker",
			Name:      "income_tasks",
			Help:      "Income tasks counter.",
		},
		[]string{"rule", "alert", "executor"},
	)

	excutedTasks := pr.NewHistogramVec(
		pr.HistogramOpts{
			Namespace: "prometheus",
			Subsystem: "alert_webhooker",
			Name:      "executed_tasks",
			Help:      "Tasks with results and duration.",
		},
		[]string{"rule", "alert", "executor", "result", "error"},
	)

	pr.MustRegister(incomeTasks)
	pr.MustRegister(excutedTasks)

	p := &PrometheusMetrics{
		incomeTasks:  incomeTasks,
		excutedTasks: excutedTasks,
	}

	return p
}

// IncomeTaskInc increments income tasks counter with given parameters.
func (p *PrometheusMetrics) IncomeTaskInc(rule, alert, executor string) {
	p.incomeTasks.WithLabelValues(rule, alert, executor).Inc()
}

// ExecutedTaskObserve observes excuted tasks histogram with given parameters.
func (p *PrometheusMetrics) ExecutedTaskObserve(rule, alert, executor, result string, err error, duration time.Duration) {
	p.excutedTasks.WithLabelValues(rule, alert, executor, result, errTextOrEmpty(err)).Observe(duration.Seconds())
}

func errTextOrEmpty(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

//go:generate mockgen -source=prometheus.go -destination=prometheus_mocks.go -package=metric doc github.com/golang/mock/gomock

type incomeTasks interface {
	WithLabelValues(lvs ...string) pr.Counter
}

type excutedTasks interface {
	WithLabelValues(lvs ...string) pr.Observer
}
