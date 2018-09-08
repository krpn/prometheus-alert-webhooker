package jenkins

import (
	"errors"
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"time"
)

const (
	paramEndpoint               = "endpoint"
	paramLogin                  = "login"
	paramPassword               = "password"
	paramJob                    = "job"
	paramParameterPrefix        = "job parameter "
	paramStateRefreshDelay      = "state_refresh_delay"
	paramSecureInterationsLimit = "secure_interations_limit"

	defaultStateRefreshDelay      = 15 * time.Second
	defaultSecureBuildDelay       = 1 * time.Second
	defaultSecureInterationsLimit = 1000
)

var requiredStringParameters = []string{
	paramEndpoint,
	paramLogin,
	paramPassword,
	paramJob,
}

//go:generate mockgen -source=jenkins.go -destination=jenkins_mocks.go -package=jenkins doc github.com/golang/mock/gomock

// Jenkins is the interface of Jenkins client.
type Jenkins interface {
	Init() (*gojenkins.Jenkins, error)
	BuildJob(name string, options ...interface{}) (int64, error)
	GetBuild(jobName string, number int64) (*gojenkins.Build, error)
	GetAllBuildIds(job string) ([]gojenkins.JobBuild, error)
}

type task struct {
	executor.TaskBase
	job                    string
	stateRefreshDelay      time.Duration
	secureInterationsLimit int
	secureBuildDelay       time.Duration
	parameters             map[string]string
	jenkins                Jenkins
}

func (task *task) ExecutorName() string {
	return "Jenkins"
}

func (task *task) ExecutorDetails() interface{} {
	if len(task.parameters) == 0 {
		return map[string]interface{}{"job": task.job}
	}

	return map[string]interface{}{
		"job":        task.job,
		"parameters": task.parameters,
	}
}

func (task *task) Fingerprint() string {
	base := task.job
	// order is important
	if len(task.parameters) > 0 {
		keys := make([]string, len(task.parameters))
		for k := range task.parameters {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			base += "," + key + task.parameters[key]
		}
	}
	return utils.MD5Hash(base)
}

func (task *task) Exec(logger *logrus.Logger) error {
	queueID, err := runJob(task.jenkins, task.job, task.parameters)
	if err != nil {
		return err
	}

	time.Sleep(task.secureBuildDelay)

	var (
		buildID int64
		job     *gojenkins.Build
		iter    int
	)
	for {
		if iter >= task.secureInterationsLimit {
			return errors.New("secure iterations limit exceed")
		}
		iter++

		time.Sleep(task.stateRefreshDelay)

		buildID, err = getBuildIDEffectively(buildID, task.jenkins, task.job, queueID)
		if err != nil {
			return err
		}

		if buildID == 0 {
			continue
		}

		job, err = task.jenkins.GetBuild(task.job, buildID)
		if err != nil {
			return err
		}

		if job.Raw.Building {
			continue
		}

		break
	}

	if job.Raw.Result != gojenkins.STATUS_SUCCESS {
		return errors.New("build failed")
	}

	return nil
}

func runJob(j Jenkins, job string, parameters map[string]string) (int64, error) {
	_, err := j.Init()
	if err != nil {
		return 0, err
	}

	return j.BuildJob(job, parameters)
}

func getBuildIDEffectively(currBuildID int64, j Jenkins, job string, queueID int64) (int64, error) {
	if currBuildID != 0 {
		return currBuildID, nil
	}

	return getBuildID(j, job, queueID)
}

func getBuildID(j Jenkins, job string, queueID int64) (int64, error) {
	builds, err := j.GetAllBuildIds(job)
	if err != nil {
		return 0, err
	}

	for _, buildID := range builds {
		build, err := j.GetBuild(job, buildID.Number)
		if err != nil {
			return 0, err
		}

		if build.Raw.QueueID == queueID {
			return buildID.Number, nil
		}
	}

	return 0, nil
}

type taskExecutor struct{}

// NewExecutor creates TaskExecutor for Jenkins tasks.
func NewExecutor() executor.TaskExecutor {
	return taskExecutor{}
}

func (executor taskExecutor) ValidateParameters(parameters map[string]interface{}) error {
	for _, reqParam := range requiredStringParameters {
		_, ok := parameters[reqParam]
		if !ok {
			return fmt.Errorf("required parameter %v is missing", reqParam)
		}
		_, ok = parameters[reqParam].(string)
		if !ok {
			return fmt.Errorf("%v parameter value is not a string", reqParam)
		}
	}

	for key, val := range parameters {
		if !strings.HasPrefix(key, paramParameterPrefix) {
			continue
		}

		if _, ok := val.(string); !ok {
			return fmt.Errorf("%v parameter value is not a string", key)
		}
	}

	return nil
}

func (executor taskExecutor) NewTask(eventID, rule, alert string, blockTTL time.Duration, preparedParameters map[string]interface{}) executor.Task {
	task := &task{
		job: preparedParameters[paramJob].(string),
	}

	task.stateRefreshDelay = defaultStateRefreshDelay
	if delayStr, ok := preparedParameters[paramStateRefreshDelay].(string); ok {
		delay, err := time.ParseDuration(delayStr)
		if err == nil {
			task.stateRefreshDelay = delay
		}
	}

	task.secureInterationsLimit = defaultSecureInterationsLimit
	if limit, ok := preparedParameters[paramSecureInterationsLimit].(int); ok && limit > 0 {
		task.secureInterationsLimit = limit
	}

	task.secureBuildDelay = defaultSecureBuildDelay

	parameters := make(map[string]string)
	for key, val := range preparedParameters {
		valStr, ok := val.(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(key, paramParameterPrefix) {
			parameters[strings.TrimSpace(strings.Replace(key, paramParameterPrefix, "", 1))] = valStr
		}
	}
	task.parameters = parameters

	task.jenkins = gojenkins.CreateJenkins(
		nil,
		preparedParameters[paramEndpoint].(string),
		preparedParameters[paramLogin],
		preparedParameters[paramPassword],
	)
	task.SetBase(eventID, rule, alert, blockTTL)
	return task
}
