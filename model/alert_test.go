package model

import (
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func TestAlert_match(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase      string
		alert      alert
		conditions Conditions
		expected   bool
	}

	testTable := []testTableData{
		{
			tcase: "status equal",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus:       "firing",
				AlertLabels:       nil,
				AlertLabelsRegexp: nil,
			},
			expected: true,
		},
		{
			tcase: "status not equal",
			alert: alert{
				Status: "resolved",
				Labels: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus:       "firing",
				AlertLabels:       nil,
				AlertLabelsRegexp: nil,
			},
			expected: false,
		},
		{
			tcase: "label equal",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: map[string]string{
					"instance": "testinstance1",
				},
				AlertLabelsRegexp: nil,
			},
			expected: true,
		},
		{
			tcase: "annotation equal",
			alert: alert{
				Status: "firing",
				Annotations: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertAnnotations: map[string]string{
					"instance": "testinstance1",
				},
				AlertLabelsRegexp: nil,
			},
			expected: true,
		},
		{
			tcase: "label not equal",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "testinstance2",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: map[string]string{
					"instance": "testinstance1",
				},
				AlertLabelsRegexp: nil,
			},
			expected: false,
		},
		{
			tcase: "label not exists",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: map[string]string{
					"job": "testjob1",
				},
				AlertLabelsRegexp: nil,
			},
			expected: false,
		},
		{
			tcase: "label regexp match",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: nil,
				AlertLabelsRegexp: map[string]*regexp.Regexp{
					"instance": regexp.MustCompile("^testinstance(.*?)"),
				},
			},
			expected: true,
		},
		{
			tcase: "label regexp not match",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "test1instance2",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: nil,
				AlertLabelsRegexp: map[string]*regexp.Regexp{
					"instance": regexp.MustCompile("^testinstance(.*?)"),
				},
			},
			expected: false,
		},
		{
			tcase: "label regexp not exists",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: nil,
				AlertLabelsRegexp: map[string]*regexp.Regexp{
					"job": regexp.MustCompile("^testjob(.*?)"),
				},
			},
			expected: false,
		},
		{
			tcase: "few conditions",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"job":      "testjob1",
					"instance": "testinstance1",
				},
			},
			conditions: Conditions{
				AlertStatus: "firing",
				AlertLabels: map[string]string{
					"job": "testjob1",
				},
				AlertLabelsRegexp: map[string]*regexp.Regexp{
					"instance": regexp.MustCompile("^testinstance(.*?)"),
				},
			},
			expected: true,
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.alert.match(testUnit.conditions), testUnit.tcase)
	}
}

func TestAlerts_ToTasksGroups(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executorMock := executor.NewMockTaskExecutor(ctrl)
	task := executor.NewMockTask(ctrl)

	type testTableData struct {
		tcase            string
		eventID          string
		alerts           Alerts
		rules            Rules
		expectFunc       func(e *executor.MockTaskExecutor)
		expectedTasksQty int
	}

	testTable := []testTableData{
		{
			tcase:   "all expect one equal",
			eventID: "998e",
			alerts: Alerts{
				{
					Status: "firing",
					Labels: map[string]string{
						"alertname":  "testalert1",
						"some_label": "value1",
					},
				},
				{
					Status: "firing",
					Labels: map[string]string{
						"alertname":  "testalert2",
						"some_label": "value2",
					},
				},
				{
					Status: "firing",
					Labels: map[string]string{
						"alertname":  "testalert3",
						"some_label": "1value3",
					},
				},
			},
			rules: Rules{
				{
					Name: "testrule1",
					Conditions: Conditions{
						AlertStatus: "firing",
						AlertLabels: nil,
						AlertLabelsRegexp: map[string]*regexp.Regexp{
							"some_label": regexp.MustCompile("^value(.*?)"),
						},
					},
					Actions: Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "some cmd",
							},
							Block:        1 * time.Second,
							TaskExecutor: executorMock,
						},
					},
				},
			},
			expectFunc: func(e *executor.MockTaskExecutor) {
				e.EXPECT().NewTask("998e", "testrule1", "testalert1", 1*time.Second, map[string]interface{}{"command": "some cmd"}).Return(task)
				e.EXPECT().NewTask("998e", "testrule1", "testalert2", 1*time.Second, map[string]interface{}{"command": "some cmd"}).Return(task)
			},
			expectedTasksQty: 2,
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(executorMock)
		assert.Equal(t, testUnit.expectedTasksQty, len(testUnit.alerts.ToTasksGroups(testUnit.rules, testUnit.eventID)), testUnit.tcase)
	}
}

func TestPrepareParams(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		alert    alert
		params   map[string]interface{}
		expected map[string]interface{}
	}

	testTable := []testTableData{
		{
			tcase: "multiply replaces",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"label1":   "value1",
					"instance": "s1.server.com:8080",
				},
				Annotations: map[string]string{
					"desc": "some description",
				},
			},
			params: map[string]interface{}{
				"command1": "${LABEL_LABEL1}",
				"url":      "https://domain.com/test?desc=${URLENCODE_ANNOTATION_DESC}",
				"instance": "${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
			},
			expected: map[string]interface{}{
				"command1": "value1",
				"url":      "https://domain.com/test?desc=some+description",
				"instance": "s1.server.com",
			},
		},
		{
			tcase: "wrong type",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"label1":   "value1",
					"instance": "s1.server.com:8080",
				},
				Annotations: map[string]string{
					"desc": "some description",
				},
			},
			params: map[string]interface{}{
				"command1": "${LABEL_LABEL1}",
				"percent":  10,
			},
			expected: map[string]interface{}{
				"command1": "value1",
				"percent":  10,
			},
		},
		{
			tcase: "placeholder in annotations",
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"label1":   "value1",
					"instance": "s1.server.com:8080",
				},
				Annotations: map[string]string{
					"desc": "https://domain.com/test?instance=${CUT_AFTER_LAST_COLON_LABEL_INSTANCE}",
				},
			},
			params: map[string]interface{}{
				"command1": "${LABEL_LABEL1}",
				"url":      "${ANNOTATION_DESC}",
			},
			expected: map[string]interface{}{
				"command1": "value1",
				"url":      "https://domain.com/test?instance=s1.server.com",
			},
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, prepareParams(testUnit.params, testUnit.alert), testUnit.tcase)
	}
}
