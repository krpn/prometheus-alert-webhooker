package runner

import (
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Start starts runners for observe tasks.
func Start(runners int, tasksCh chan model.Tasks, blocker blocker, metric metricser, logger *logrus.Logger, nowFunc func() time.Time) {
	var wg sync.WaitGroup
	wg.Add(runners)
	for i := 0; i < runners; i++ {
		go runner(tasksCh, blocker, metric, logger, nowFunc, &wg)
	}
	wg.Wait()
}

const context = "runner"

func runner(tasksCh chan model.Tasks, blocker blocker, metric metricser, logger *logrus.Logger, nowFunc func() time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	var (
		result    execResult
		err       error
		start     time.Time
		ctxLogger = logger.WithField("context", context)
	)

	for tasks := range tasksCh {
		tasksLogger := ctxLogger.WithField("tasks", tasks.Details())
		tasksLogger.Debug("runner starts executing group")

		tasksQty := len(tasks)
		for i, task := range tasks {
			taskNum := i + 1
			taskLogger := tasksLogger.WithFields(executor.TaskDetails(task))
			taskLogger.Debugf("runner starts executing task #%v/%v", taskNum, tasksQty)

			start = nowFunc()
			result, err = exec(task, blocker, logger)
			duration := nowFunc().Sub(start)
			metric.ExecutedTaskObserve(task.Rule(), task.Alert(), task.ExecutorName(), result.String(), err, duration)

			taskLogger = taskLogger.WithFields(logrus.Fields{"result": result.String(), "duration": duration.String()})
			if err == nil {
				taskLogger.Debugf("runner finished executing task #%v/%v", taskNum, tasksQty)
			} else {
				taskLogger.Errorf("runner got executing task #%v/%v error, stopping group: %v", taskNum, tasksQty, err)
				break
			}

			if !utils.StringSliceContains(successfulResults, string(result)) {
				taskLogger.Debugf("runner got executing task #%v/%v unsuccessful result, stopping group: %v", taskNum, tasksQty, result)
				break
			}
		}

		tasksLogger.Debug("runner finished executing group")
	}
}

//go:generate mockgen -source=runner.go -destination=runner_mocks.go -package=runner doc github.com/golang/mock/gomock

type blocker interface {
	BlockInProgress(executor, fingerprint string) (blockedSuccessfully bool, err error)
	BlockForTTL(executor, fingerprint string, ttl time.Duration) (err error)
	Unblock(executor, fingerprint string)
}

type metricser interface {
	ExecutedTaskObserve(rule, alert, executor, result string, err error, duration time.Duration)
}
