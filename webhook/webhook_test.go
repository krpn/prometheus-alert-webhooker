package webhook

import (
	"bytes"
	"github.com/golang/mock/gomock"
	"github.com/jinzhu/copier"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/model"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestWebhook(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	metric := NewMockmetricser(ctrl)
	executorMock := executor.NewMockTaskExecutor(ctrl)
	task := executor.NewMockTask(ctrl)

	nowFunc := func() time.Time {
		return time.Unix(1535086351, 0)
	}

	type testTableData struct {
		tcase         string
		rules         model.Rules
		body          []byte
		expectFunc    func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask)
		expectedTasks model.TasksGroups
		expectedLogs  []string
	}

	testTable := []testTableData{
		{
			tcase: "correct alert",
			rules: []model.Rule{
				{
					Name: "testrule1",
					Conditions: model.Conditions{
						AlertStatus: "firing",
						AlertLabels: map[string]string{
							"instance": "testinstance1",
						},
					},
					Actions: model.Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "curl ${ANNOTATION_URL}",
							},
							Block:        1 * time.Minute,
							TaskExecutor: executorMock,
						},
					},
				},
			},
			body: []byte(`{
    "alerts": [
        {
            "annotations": {
                "url": "http://jenkins.../job?val=${LABEL_INSTANCE}"
            },
            "labels": {
                "alertname": "testalert1",
                "instance": "testinstance1"
            }
        }
    ],
    "status": "firing"
}`),
			expectFunc: func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask) {
				e.EXPECT().NewTask("dc12", "testrule1", "testalert1", 1*time.Minute, map[string]interface{}{
					"command": "curl http://jenkins.../job?val=testinstance1",
				}).Return(t)
				t.EXPECT().EventID().Return("dc12").Times(2)
				t.EXPECT().Rule().Return("testrule1").Times(3)
				t.EXPECT().Alert().Return("testalert1").Times(3)
				t.EXPECT().ExecutorName().Return("shell").Times(3)
				t.EXPECT().ExecutorDetails().Return(map[string]interface{}{
					"command": "curl http://jenkins.../job?val=testinstance1",
				}).Times(2)
				m.EXPECT().IncomeTaskInc("testrule1", "testalert1", "shell")
			},
			expectedTasks: model.TasksGroups{{task}},
			expectedLogs: []string{
				`{"context":"webhook","event_id":"dc12","level":"debug","msg":"payload is received, tasks are prepared","payload":{"receiver":"","status":"firing","alerts":[{"status":"","labels":{"alertname":"testalert1","instance":"testinstance1"},"annotations":{"url":"http://jenkins.../job?val=${LABEL_INSTANCE}"},"startsAt":"0001-01-01T00:00:00Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":null,"commonLabels":null,"commonAnnotations":null,"externalURL":""},"tasks_groups":[[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]]}`,
				`{"context":"webhook","level":"debug","msg":"ready to send tasks to runner","tasks":[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]}`,
				`{"context":"webhook","level":"debug","msg":"sent tasks to runner","tasks":[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]}`,
				`{"context":"webhook","event_id":"dc12","level":"debug","msg":"all tasks sent to runners","payload":{"receiver":"","status":"firing","alerts":[{"status":"","labels":{"alertname":"testalert1","instance":"testinstance1"},"annotations":{"url":"http://jenkins.../job?val=${LABEL_INSTANCE}"},"startsAt":"0001-01-01T00:00:00Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":null,"commonLabels":null,"commonAnnotations":null,"externalURL":""},"tasks_groups":[[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]]}`,
			},
		},
		{
			tcase:        "empty request",
			body:         nil,
			expectFunc:   func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask) {},
			expectedLogs: []string{},
		},
		{
			tcase: "no tasks for payload",
			rules: []model.Rule{
				{
					Name: "testrule1",
					Conditions: model.Conditions{
						AlertStatus: "firing",
						AlertLabels: map[string]string{
							"instance": "testinstance1",
						},
					},
					Actions: model.Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "curl ${ANNOTATION_URL}",
							},
							Block:        1 * time.Minute,
							TaskExecutor: executorMock,
						},
					},
				},
			},
			body: []byte(`{
    "alerts": [
        {
            "annotations": {
                "url": "http://jenkins.../job?val=${LABEL_INSTANCE}"
            },
            "labels": {
                "alertname": "testalert1",
                "instance": "testinstance2"
            }
        }
    ],
    "status": "firing"
}`),
			expectFunc:    func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask) {},
			expectedTasks: model.TasksGroups{},
			expectedLogs: []string{
				`{"context":"webhook","event_id":"dc12","level":"debug","msg":"payload is received, no tasks for it","payload":{"receiver":"","status":"firing","alerts":[{"status":"","labels":{"alertname":"testalert1","instance":"testinstance2"},"annotations":{"url":"http://jenkins.../job?val=${LABEL_INSTANCE}"},"startsAt":"0001-01-01T00:00:00Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":null,"commonLabels":null,"commonAnnotations":null,"externalURL":""},"tasks_groups":[]}`,
			},
		},
	}

	for _, testUnit := range testTable {
		logger, hook := test.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		logger.Formatter = &logrus.JSONFormatter{DisableTimestamp: true}

		testUnit.expectFunc(metric, executorMock, task)

		req, err := http.NewRequest("POST", "http://prometheus-alert-webhooker.com/", bytes.NewBuffer(testUnit.body))
		if err != nil {
			t.Fatal(err)
		}

		tasksCh := make(chan model.Tasks, len(testUnit.expectedTasks))
		Webhook(req, testUnit.rules, tasksCh, metric, logger, nowFunc)

		for _, expectedTask := range testUnit.expectedTasks {
			assert.Equal(t, expectedTask, <-tasksCh, testUnit.tcase)
		}

		assert.Equal(t, expectedLogsFix(testUnit.expectedLogs), logsFromHook(t, hook), testUnit.tcase)
	}
}

func TestWebhook_RulesChanged(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	metric := NewMockmetricser(ctrl)
	executorMock := executor.NewMockTaskExecutor(ctrl)
	task := executor.NewMockTask(ctrl)

	nowFunc := func() time.Time {
		return time.Unix(1535086351, 0)
	}

	body := []byte(`{
    "alerts": [
        {
            "annotations": {
                "url": "http://jenkins.../job?val=${LABEL_INSTANCE}"
            },
            "labels": {
                "alertname": "testalert1",
                "instance": "testinstance1"
            }
        }
    ],
    "status": "firing"
}`)

	type testTableData struct {
		tcase         string
		rules         model.Rules
		expectFunc    func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask)
		expectedTasks model.TasksGroups
		expectedLogs  []string
	}

	testTable := []testTableData{
		{
			tcase: "cross",
			rules: []model.Rule{
				{
					Name: "testrule1",
					Conditions: model.Conditions{
						AlertStatus: "firing",
						AlertLabels: map[string]string{
							"instance": "testinstance1",
						},
					},
					Actions: model.Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "curl ${ANNOTATION_URL}",
							},
							Block:        1 * time.Minute,
							TaskExecutor: executorMock,
						},
					},
				},
			},
			expectFunc: func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask) {
				e.EXPECT().NewTask("dc12", "testrule1", "testalert1", 1*time.Minute, map[string]interface{}{
					"command": "curl http://jenkins.../job?val=testinstance1",
				}).Return(t)
				t.EXPECT().EventID().Return("dc12").Times(2)
				t.EXPECT().Rule().Return("testrule1").Times(3)
				t.EXPECT().Alert().Return("testalert1").Times(3)
				t.EXPECT().ExecutorName().Return("shell").Times(3)
				t.EXPECT().ExecutorDetails().Return(map[string]interface{}{
					"command": "curl http://jenkins.../job?val=testinstance1",
				}).Times(2)
				m.EXPECT().IncomeTaskInc("testrule1", "testalert1", "shell")
			},
			expectedTasks: model.TasksGroups{{task}},
			expectedLogs: []string{
				`{"context":"webhook","event_id":"dc12","level":"debug","msg":"payload is received, tasks are prepared","payload":{"receiver":"","status":"firing","alerts":[{"status":"","labels":{"alertname":"testalert1","instance":"testinstance1"},"annotations":{"url":"http://jenkins.../job?val=${LABEL_INSTANCE}"},"startsAt":"0001-01-01T00:00:00Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":null,"commonLabels":null,"commonAnnotations":null,"externalURL":""},"tasks_groups":[[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]]}`,
				`{"context":"webhook","level":"debug","msg":"ready to send tasks to runner","tasks":[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]}`,
				`{"context":"webhook","level":"debug","msg":"sent tasks to runner","tasks":[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]}`,
				`{"context":"webhook","event_id":"dc12","level":"debug","msg":"all tasks sent to runners","payload":{"receiver":"","status":"firing","alerts":[{"status":"","labels":{"alertname":"testalert1","instance":"testinstance1"},"annotations":{"url":"http://jenkins.../job?val=${LABEL_INSTANCE}"},"startsAt":"0001-01-01T00:00:00Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":null,"commonLabels":null,"commonAnnotations":null,"externalURL":""},"tasks_groups":[[{"alert":"testalert1","details":{"command":"curl http://jenkins.../job?val=testinstance1"},"event_id":"dc12","executor":"shell","rule":"testrule1"}]]}`,
			},
		},
		{
			tcase: "miss",
			rules: []model.Rule{
				{
					Name: "testrule1",
					Conditions: model.Conditions{
						AlertStatus: "firing",
						AlertLabels: map[string]string{
							"instance": "testinstance2",
						},
					},
					Actions: model.Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "curl ${ANNOTATION_URL}",
							},
							Block:        1 * time.Minute,
							TaskExecutor: executorMock,
						},
					},
				},
			},
			expectFunc:    func(m *Mockmetricser, e *executor.MockTaskExecutor, t *executor.MockTask) {},
			expectedTasks: model.TasksGroups{},
			expectedLogs: []string{
				`{"context":"webhook","event_id":"dc12","level":"debug","msg":"payload is received, no tasks for it","payload":{"receiver":"","status":"firing","alerts":[{"status":"","labels":{"alertname":"testalert1","instance":"testinstance1"},"annotations":{"url":"http://jenkins.../job?val=${LABEL_INSTANCE}"},"startsAt":"0001-01-01T00:00:00Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":null,"commonLabels":null,"commonAnnotations":null,"externalURL":""},"tasks_groups":[]}`,
			},
		},
	}

	var globalRules model.Rules

	for _, testUnit := range testTable {
		assert.NoError(t, copier.Copy(&globalRules, &testUnit.rules), testUnit.tcase)

		logger, hook := test.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		logger.Formatter = &logrus.JSONFormatter{DisableTimestamp: true}

		testUnit.expectFunc(metric, executorMock, task)

		req, err := http.NewRequest("POST", "http://prometheus-alert-webhooker.com/", bytes.NewBuffer(body))
		if err != nil {
			t.Fatal(err)
		}

		tasksCh := make(chan model.Tasks, len(testUnit.expectedTasks))
		Webhook(req, globalRules, tasksCh, metric, logger, nowFunc)

		for _, expectedTask := range testUnit.expectedTasks {
			assert.Equal(t, expectedTask, <-tasksCh, testUnit.tcase)
		}

		assert.Equal(t, expectedLogsFix(testUnit.expectedLogs), logsFromHook(t, hook), testUnit.tcase)
	}
}

func TestGetEventID(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		nowFunc  func() time.Time
		expected string
	}

	testTable := []testTableData{
		{
			nowFunc: func() time.Time {
				return time.Unix(1535086351, 0)
			},
			expected: "dc12",
		},
		{
			nowFunc: func() time.Time {
				return time.Unix(1535139645, 0)
			},
			expected: "a294",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, getEventID(testUnit.nowFunc))
	}
}

func logsFromHook(t *testing.T, hook *test.Hook) (logs []string) {
	if hook == nil {
		return []string{}
	}

	if hook.Entries == nil {
		return []string{}
	}

	logs = make([]string, len(hook.Entries))
	for i, entry := range hook.Entries {
		log, err := entry.String()
		assert.Equal(t, nil, err)
		logs[i] = log
	}
	return
}

func expectedLogsFix(logs []string) (expectedLogs []string) {
	expectedLogs = make([]string, len(logs))
	for i, log := range logs {
		expectedLogs[i] = log + "\n"
	}
	return
}
