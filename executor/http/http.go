package httpe

import (
	"bytes"
	"fmt"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	paramMethod            = "method"
	paramURL               = "url"
	paramBody              = "body"
	paramHeaderPrefix      = "header "
	paramTimeout           = "timeout"
	paramSuccessHTTPStatus = "success_http_status"

	defaultMethod            = http.MethodGet
	defaultTimeout           = 1 * time.Second
	defaultSuccessHTTPStatus = http.StatusOK
)

var stringParameters = []string{
	paramMethod,
	paramURL,
	paramBody,
}

//go:generate mockgen -source=http.go -destination=http_mocks.go -package=httpe doc github.com/golang/mock/gomock

// Doer is the interface of HTTP client.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type task struct {
	executor.TaskBase
	method            string
	url               string
	body              string
	headers           map[string]string
	successHTTPStatus int
	client            Doer
}

func (task *task) ExecutorName() string {
	return "http"
}

func (task *task) ExecutorDetails() interface{} {
	d := map[string]interface{}{
		"method": task.method,
		"url":    task.url,
	}

	if task.body != "" {
		d["body"] = task.body
	}

	if len(task.headers) > 0 {
		d["headers"] = task.headers
	}

	return d
}

func (task *task) Fingerprint() string {
	base := fmt.Sprintf("%v|%v|%v", task.method, task.url, task.body)
	// order is important
	if len(task.headers) > 0 {
		keys := make([]string, len(task.headers))
		for k := range task.headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			base += "," + key + task.headers[key]
		}
	}
	return utils.MD5Hash(base)
}

func (task *task) Exec(logger *logrus.Logger) error {
	var (
		req *http.Request
		err error
	)
	if task.body == "" {
		req, err = http.NewRequest(task.method, task.url, nil)
	} else {
		req, err = http.NewRequest(task.method, task.url, bytes.NewBufferString(task.body))
	}
	if err != nil {
		return err
	}

	for key, val := range task.headers {
		req.Header.Set(key, val)
	}

	resp, err := task.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != task.successHTTPStatus {
		return fmt.Errorf("returned HTTP status: %v, body close error: %v", resp.StatusCode, resp.Body.Close())
	}

	return resp.Body.Close()
}

type taskExecutor struct {
	clientGen func(time.Duration) Doer
}

// NewExecutor creates TaskExecutor for HTTP tasks.
func NewExecutor(clientGen func(time.Duration) Doer) executor.TaskExecutor {
	return taskExecutor{clientGen: clientGen}
}

func (executor taskExecutor) ValidateParameters(parameters map[string]interface{}) error {
	if _, ok := parameters[paramURL]; !ok {
		return fmt.Errorf("required parameter %v is missing", paramURL)
	}

	for _, reqParam := range stringParameters {
		if _, ok := parameters[reqParam]; ok {
			if _, ok := parameters[reqParam].(string); !ok {
				return fmt.Errorf("%v parameter value is not a string", reqParam)
			}
		}
	}

	for key, val := range parameters {
		if !strings.HasPrefix(key, paramHeaderPrefix) {
			continue
		}

		if _, ok := val.(string); !ok {
			return fmt.Errorf("%v parameter value is not a string", key)
		}
	}

	return nil
}

func (executor taskExecutor) NewTask(eventID, rule, alert string, blockTTL time.Duration, preparedParameters map[string]interface{}) executor.Task {
	method := defaultMethod
	if m, ok := preparedParameters[paramMethod]; ok {
		if ms, ok := m.(string); ok {
			method = ms
		}
	}

	task := &task{
		method: method,
		url:    preparedParameters[paramURL].(string),
	}

	task.body, _ = preparedParameters[paramBody].(string)

	headers := make(map[string]string)
	for key, val := range preparedParameters {
		valStr, ok := val.(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(key, paramHeaderPrefix) {
			headers[strings.TrimSpace(strings.Replace(key, paramHeaderPrefix, "", 1))] = valStr
		}
	}
	task.headers = headers

	timeout := defaultTimeout
	if timeoutStr, ok := preparedParameters[paramTimeout].(string); ok {
		tm, err := time.ParseDuration(timeoutStr)
		if err == nil {
			timeout = tm
		}
	}

	task.successHTTPStatus = defaultSuccessHTTPStatus
	if status, ok := preparedParameters[paramSuccessHTTPStatus].(int); ok && status > 0 {
		task.successHTTPStatus = status
	}

	task.client = executor.clientGen(timeout)

	task.SetBase(eventID, rule, alert, blockTTL)
	return task
}
