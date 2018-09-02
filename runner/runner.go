package runner

import (
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Start starts runners for observe tasks.
func Start(runners int, tasksCh chan executor.Task, blocker blocker, metric metricser, logger *logrus.Logger, nowFunc func() time.Time) {
	var wg sync.WaitGroup
	wg.Add(runners)
	for i := 0; i < runners; i++ {
		go runner(tasksCh, blocker, metric, logger, nowFunc, &wg)
	}
	wg.Wait()
}

const context = "runner"

func runner(tasksCh chan executor.Task, blocker blocker, metric metricser, logger *logrus.Logger, nowFunc func() time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	var (
		result    execResult
		err       error
		start     time.Time
		ctxLogger = logger.WithField("context", context)
	)

	for task := range tasksCh {
		taskLogger := ctxLogger.WithFields(executor.TaskDetails(task))
		taskLogger.Info("runner starts executing")

		start = nowFunc()
		result, err = exec(task, blocker, logger)
		duration := nowFunc().Sub(start)

		taskLogger = taskLogger.WithFields(logrus.Fields{"result": result.String(), "duration": duration.String()})
		if err == nil {
			taskLogger.Info("runner finished executing")
		} else {
			taskLogger.Errorf("runner got executing error: %v", err)
		}

		metric.ExecutedTaskObserve(task.Rule(), task.Alert(), task.ExecutorName(), result.String(), err, duration)
	}
}

//go:generate mockgen -source=runner.go -destination=runner_mocks.go -package=runner doc github.com/golang/mock/gomock

type blocker interface {
	BlockInProgress(fingerprint string) (blockedSuccessfully bool, err error)
	BlockForTTL(fingerprint string, ttl time.Duration) (err error)
	Unblock(fingerprint string)
}

type metricser interface {
	ExecutedTaskObserve(rule, alert, executor, result string, err error, duration time.Duration)
}
