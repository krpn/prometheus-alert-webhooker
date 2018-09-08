package httpe

import (
	"bytes"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
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
				"success_http_status":  200,
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
				"success_http_status":  200,
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
		{
			tcase: "param header Int wrong type",
			params: map[string]interface{}{
				"url":                  "http://www.test.com/",
				"method":               "POST",
				"body":                 "some body",
				"header Authorization": "ba0828c9fac6b0b47d9147963429d091",
				"header Int":           123,
				"timeout":              "10s",
				"success_http_status":  200,
			},
			expected: errors.New("header Int parameter value is not a string"),
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
					method:            "GET",
					url:               "http://www.test.com/",
					body:              "",
					headers:           map[string]string{},
					successHTTPStatus: defaultSuccessHTTPStatus,
					client:            &http.Client{Timeout: 1 * time.Second},
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
				"success_http_status":      201,
			},
			expected: func() executor.Task {
				task := &task{
					method:            "POST",
					url:               "http://www.test.com/",
					body:              "some body",
					headers:           map[string]string{"Authorization": "ba0828c9fac6b0b47d9147963429d091"},
					successHTTPStatus: 201,
					client:            &http.Client{Timeout: 10 * time.Second},
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

func TestHTTPTask_Exec(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	doerMock := NewMockDoer(ctrl)

	logger, hook := test.NewNullLogger()

	type testTableData struct {
		tcase       string
		task        func() *task
		expectFunc  func(t *MockDoer)
		expectedErr error
	}

	testTable := []testTableData{
		{
			tcase: "success with minimal parameters",
			task: func() *task {
				task := &task{
					method:            "GET",
					url:               "http://www.test.com/",
					body:              "",
					headers:           map[string]string{},
					successHTTPStatus: http.StatusOK,
					client:            doerMock,
				}
				task.SetBase("id", "rule", "alert", 10*time.Minute)
				return task
			},
			expectFunc: func(t *MockDoer) {
				req, _ := http.NewRequest("GET", "http://www.test.com/", nil)
				t.EXPECT().Do(req).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewBufferString("resp body")),
				}, nil)
			},
			expectedErr: nil,
		},
		{
			tcase: "bad status code",
			task: func() *task {
				task := &task{
					method:            "GET",
					url:               "http://www.test.com/",
					body:              "",
					headers:           map[string]string{},
					successHTTPStatus: http.StatusOK,
					client:            doerMock,
				}
				task.SetBase("id", "rule", "alert", 10*time.Minute)
				return task
			},
			expectFunc: func(t *MockDoer) {
				req, _ := http.NewRequest("GET", "http://www.test.com/", nil)
				t.EXPECT().Do(req).Return(&http.Response{
					StatusCode: http.StatusGatewayTimeout,
					Body:       ioutil.NopCloser(bytes.NewBufferString("resp body")),
				}, nil)
			},
			expectedErr: errors.New("returned HTTP status: 504, body close error: <nil>"),
		},
		{
			tcase: "request error",
			task: func() *task {
				task := &task{
					method:            "GET",
					url:               "http://www.test.com/",
					body:              "",
					headers:           map[string]string{},
					successHTTPStatus: http.StatusOK,
					client:            doerMock,
				}
				task.SetBase("id", "rule", "alert", 10*time.Minute)
				return task
			},
			expectFunc: func(t *MockDoer) {
				req, _ := http.NewRequest("GET", "http://www.test.com/", nil)
				t.EXPECT().Do(req).Return(nil, errors.New("request error"))
			},
			expectedErr: errors.New("request error"),
		},
		{
			tcase: "success with header parameter",
			task: func() *task {
				task := &task{
					method:            "GET",
					url:               "http://www.test.com/",
					body:              "",
					headers:           map[string]string{"Authorization": "ba0828c9fac6b0b47d9147963429d091"},
					successHTTPStatus: http.StatusOK,
					client:            doerMock,
				}
				task.SetBase("id", "rule", "alert", 10*time.Minute)
				return task
			},
			expectFunc: func(t *MockDoer) {
				req, _ := http.NewRequest("GET", "http://www.test.com/", nil)
				req.Header.Set("Authorization", "ba0828c9fac6b0b47d9147963429d091")
				t.EXPECT().Do(req).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewBufferString("resp body")),
				}, nil)
			},
			expectedErr: nil,
		},
		{
			tcase: "new request error",
			task: func() *task {
				task := &task{
					method:            "POST",
					url:               "http://www test com/",
					body:              "some body",
					headers:           map[string]string{"Authorization": "ba0828c9fac6b0b47d9147963429d091"},
					successHTTPStatus: http.StatusOK,
					client:            doerMock,
				}
				task.SetBase("id", "rule", "alert", 10*time.Minute)
				return task
			},
			expectFunc:  func(t *MockDoer) {},
			expectedErr: &url.Error{Op: "parse", URL: "http://www test com/", Err: url.InvalidHostError(" ")},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(doerMock)
		assert.Equal(t, testUnit.expectedErr, testUnit.task().Exec(logger), testUnit.tcase)
	}

	// logger is not used
	assert.Equal(t, 0, len(hook.Entries))
}
