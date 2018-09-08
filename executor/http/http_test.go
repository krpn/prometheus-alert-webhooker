package httpe

import (
	"errors"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestHTTPTask_ExecutorInterface(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(func(timeout time.Duration) Doer {
		return &http.Client{Timeout: timeout}
	})

	type testTableData struct {
		tcase    string
		task     func() executor.Task
		taskFunc func(t executor.Task) interface{}
		expected interface{}
	}

	testTable := []testTableData{
		{
			tcase: "ExecutorName func",
			task: func() executor.Task {
				return executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
					map[string]interface{}{
						"url": "http://www.test.com/",
					},
				)
			},
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorName()
			},
			expected: "http",
		},
		{
			tcase: "ExecutorDetails func",
			task: func() executor.Task {
				return executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
					map[string]interface{}{
						"url": "http://www.test.com/",
					},
				)
			},
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{"method": "GET", "url": "http://www.test.com/"},
		},
		{
			tcase: "ExecutorDetails func + headers",
			task: func() executor.Task {
				return executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
					map[string]interface{}{
						"url":                  "http://www.test.com/",
						"header Authorization": "ba0828c9fac6b0b47d9147963429d091",
					},
				)
			},
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{
				"method": "GET",
				"url":    "http://www.test.com/",
				"headers": map[string]string{
					"Authorization": "ba0828c9fac6b0b47d9147963429d091",
				},
			},
		},
		{
			tcase: "ExecutorDetails func + body",
			task: func() executor.Task {
				return executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
					map[string]interface{}{
						"url":  "http://www.test.com/",
						"body": "some body",
					},
				)
			},
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{
				"method": "GET",
				"url":    "http://www.test.com/",
				"body":   "some body",
			},
		},
		{
			tcase: "Fingerprint func",
			task: func() executor.Task {
				return executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
					map[string]interface{}{
						"url": "http://www.test.com/",
					},
				)
			},
			taskFunc: func(t executor.Task) interface{} {
				return t.Fingerprint()
			},
			expected: "ba0828c9fac6b0b47d9147963429d091",
		},
		{
			tcase: "Fingerprint func + headers",
			task: func() executor.Task {
				return executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
					map[string]interface{}{
						"url":                  "http://www.test.com/",
						"header Authorization": "ba0828c9fac6b0b47d9147963429d091",
					},
				)
			},
			taskFunc: func(t executor.Task) interface{} {
				return t.Fingerprint()
			},
			expected: "192789786d1fb9cfdfc2b9f207f50b7c",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.taskFunc(testUnit.task()), testUnit.tcase)
	}
}

func TestHTTPTaskExecutor_ValidateParameters(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(func(timeout time.Duration) Doer {
		return &http.Client{Timeout: timeout}
	})

	type testTableData struct {
		tcase    string
		params   map[string]interface{}
		expected error
	}

	testTable := []testTableData{
		{
			tcase: "correct minimal params",
			params: map[string]interface{}{
				"url": "http://www.test.com/",
			},
			expected: nil,
		},
		{
			tcase: "correct maximal params",
			params: map[string]interface{}{
				"url":                  "http://www.test.com/",
				"method":               "POST",
				"body":                 "some body",
				"header Authorization": "ba0828c9fac6b0b47d9147963429d091",
				"timeout":              "10s",
			},
			expected: nil,
		},
		{
			tcase: "param url missing",
			params: map[string]interface{}{
				"method":               "POST",
				"body":                 "some body",
				"header Authorization": "ba0828c9fac6b0b47d9147963429d091",
				"timeout":              "10s",
			},
			expected: errors.New("required parameter url is missing"),
		},
		{
			tcase: "param body wrong type",
			params: map[string]interface{}{
				"url":    "http://www.test.com/",
				"method": "POST",
				"body":   123,
			},
			expected: errors.New("body parameter value is not a string"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, executorMock.ValidateParameters(testUnit.params), testUnit.tcase)
	}
}

func TestHTTPTaskExecutor_NewTask(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(func(timeout time.Duration) Doer {
		return &http.Client{Timeout: timeout}
	})

	type testTableData struct {
		tcase                string
		eventID, rule, alert string
		blockTTL             time.Duration
		preparedParameters   map[string]interface{}
		expected             func() executor.Task
	}

	testTable := []testTableData{
		{
			tcase:    "minimal params",
			eventID:  "825e",
			rule:     "testrule1",
			alert:    "testalert1",
			blockTTL: 1 * time.Second,
			preparedParameters: map[string]interface{}{
				"url": "http://www.test.com/",
			},
			expected: func() executor.Task {
				task := &task{
					method:  "GET",
					url:     "http://www.test.com/",
					body:    "",
					headers: map[string]string{},
					client:  &http.Client{Timeout: 1 * time.Second},
				}
				task.SetBase("825e", "testrule1", "testalert1", 1*time.Second)
				return task
			},
		},
		{
			tcase:    "all params",
			eventID:  "825e",
			rule:     "testrule1",
			alert:    "testalert1",
			blockTTL: 1 * time.Second,
			preparedParameters: map[string]interface{}{
				"method":                   "POST",
				"url":                      "http://www.test.com/",
				"body":                     "some body",
				"header Authorization":     "ba0828c9fac6b0b47d9147963429d091",
				"header Wrong type header": 123,
				"timeout":                  "10s",
			},
			expected: func() executor.Task {
				task := &task{
					method:  "POST",
					url:     "http://www.test.com/",
					body:    "some body",
					headers: map[string]string{"Authorization": "ba0828c9fac6b0b47d9147963429d091"},
					client:  &http.Client{Timeout: 10 * time.Second},
				}
				task.SetBase("825e", "testrule1", "testalert1", 1*time.Second)
				return task
			},
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected(), executorMock.NewTask(testUnit.eventID, testUnit.rule, testUnit.alert, testUnit.blockTTL, testUnit.preparedParameters), testUnit.tcase)
	}
}
