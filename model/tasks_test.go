package model

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewTasks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executorMock := executor.NewMockTaskExecutor(ctrl)
	task := executor.NewMockTask(ctrl)

	type testTableData struct {
		tcase      string
		eventID    string
		rule       func() Rule
		alert      alert
		expectFunc func(e *executor.MockTaskExecutor)
		expected   Tasks
	}

	testTable := []testTableData{
		{
			tcase:   "shell executor",
			eventID: "4a72",
			rule: func() Rule {
				rule := *getTestRuleCompiled(1)
				rule.Actions = Actions{
					{
						Executor: "shell",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block:        1 * time.Second,
						TaskExecutor: executorMock,
					},
				}
				return rule
			},
			alert: alert{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "testalert1",
					"block":     "marshaller function",
					"error":     "unmarshal error&",
					"instance":  "server.domain.com:9090",
				},
				Annotations: map[string]string{
					"title": "instance down",
				},
			},
			expectFunc: func(e *executor.MockTaskExecutor) {
				e.EXPECT().NewTask("4a72", "testrule1", "testalert1", 1*time.Second, map[string]interface{}{
					"command": "marshaller function | unmarshal+error%26 | server.domain.com | instance down",
				}).Return(task)
			},
			expected: Tasks{
				task,
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(executorMock)
		assert.Equal(t, testUnit.expected, NewTasks(testUnit.rule(), testUnit.alert, testUnit.eventID), testUnit.tcase)
	}
}

func TestTasks_Details(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type testTableData struct {
		tasks      []*executor.MockTask
		expectFunc func(t []*executor.MockTask)
		expected   []map[string]interface{}
	}

	testTable := []testTableData{
		{
			tasks: []*executor.MockTask{executor.NewMockTask(ctrl), executor.NewMockTask(ctrl)},
			expectFunc: func(ta []*executor.MockTask) {
				for i, t := range ta {
					j := i + 1
					t.EXPECT().EventID().Return(fmt.Sprintf("testeventid%v", j))
					t.EXPECT().Rule().Return(fmt.Sprintf("testrule%v", j))
					t.EXPECT().Alert().Return(fmt.Sprintf("testaler%v", j))
					t.EXPECT().ExecutorName().Return(fmt.Sprintf("testexecutor%v", j))
					t.EXPECT().ExecutorDetails().Return(map[string]string{
						"cmd": "curl some url",
					})
				}
			},
			expected: []map[string]interface{}{
				{
					"event_id": "testeventid1",
					"rule":     "testrule1",
					"alert":    "testaler1",
					"executor": "testexecutor1",
					"details": map[string]string{
						"cmd": "curl some url",
					},
				},
				{
					"event_id": "testeventid2",
					"rule":     "testrule2",
					"alert":    "testaler2",
					"executor": "testexecutor2",
					"details": map[string]string{
						"cmd": "curl some url",
					},
				},
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(testUnit.tasks)
		tasks := make(Tasks, len(testUnit.tasks))
		for i, task := range testUnit.tasks {
			tasks[i] = task
		}
		assert.Equal(t, testUnit.expected, tasks.Details())
	}
}

func TestTasksGroups_Details(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type testTableData struct {
		tasksGroups [][]*executor.MockTask
		expectFunc  func(t [][]*executor.MockTask)
		expected    [][]map[string]interface{}
	}

	testTable := []testTableData{
		{
			tasksGroups: [][]*executor.MockTask{{executor.NewMockTask(ctrl), executor.NewMockTask(ctrl)}},
			expectFunc: func(ta [][]*executor.MockTask) {
				for _, ta := range ta {
					for i, t := range ta {
						j := i + 1
						t.EXPECT().EventID().Return(fmt.Sprintf("testeventid%v", j))
						t.EXPECT().Rule().Return(fmt.Sprintf("testrule%v", j))
						t.EXPECT().Alert().Return(fmt.Sprintf("testaler%v", j))
						t.EXPECT().ExecutorName().Return(fmt.Sprintf("testexecutor%v", j))
						t.EXPECT().ExecutorDetails().Return(map[string]string{
							"cmd": "curl some url",
						})
					}
				}
			},
			expected: [][]map[string]interface{}{
				{
					{
						"event_id": "testeventid1",
						"rule":     "testrule1",
						"alert":    "testaler1",
						"executor": "testexecutor1",
						"details": map[string]string{
							"cmd": "curl some url",
						},
					},
					{
						"event_id": "testeventid2",
						"rule":     "testrule2",
						"alert":    "testaler2",
						"executor": "testexecutor2",
						"details": map[string]string{
							"cmd": "curl some url",
						},
					},
				},
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(testUnit.tasksGroups)
		tasksGroups := make(TasksGroups, len(testUnit.tasksGroups))
		for i, tasksGroup := range testUnit.tasksGroups {
			tasks := make(Tasks, len(tasksGroup))
			for j, task := range tasksGroup {
				tasks[j] = task
			}
			tasksGroups[i] = tasks
		}
		assert.Equal(t, testUnit.expected, tasksGroups.Details())
	}
}
