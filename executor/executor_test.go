package executor

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTaskDetails(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type testTableData struct {
		task       *MockTask
		expectFunc func(t *MockTask)
		expected   map[string]interface{}
	}

	testTable := []testTableData{
		{
			task: NewMockTask(ctrl),
			expectFunc: func(t *MockTask) {
				t.EXPECT().EventID().Return("testeventid1")
				t.EXPECT().Rule().Return("testrule1")
				t.EXPECT().Alert().Return("testaler1")
				t.EXPECT().ExecutorName().Return("testexecutor1")
				t.EXPECT().ExecutorDetails().Return(map[string]string{
					"cmd": "curl some url",
				})
			},
			expected: map[string]interface{}{
				"event_id": "testeventid1",
				"rule":     "testrule1",
				"alert":    "testaler1",
				"executor": "testexecutor1",
				"details": map[string]string{
					"cmd": "curl some url",
				},
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(testUnit.task)
		assert.Equal(t, testUnit.expected, TaskDetails(testUnit.task))
	}
}

// Test mock for coverage
func TestMockTaskCoverage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger, _ := test.NewNullLogger()

	type testTableData struct {
		tcase      string
		task       *MockTask
		taskFunc   func(t Task) interface{}
		expectFunc func(t *MockTask)
		expected   interface{}
	}

	testTable := []testTableData{
		{
			tcase: "Exec func",
			task:  NewMockTask(ctrl),
			taskFunc: func(t Task) interface{} {
				return t.Exec(logger)
			},
			expectFunc: func(t *MockTask) {
				t.EXPECT().Exec(logger).Return(errors.New("exec error"))
			},
			expected: errors.New("exec error"),
		},
		{
			tcase: "Fingerprint func",
			task:  NewMockTask(ctrl),
			taskFunc: func(t Task) interface{} {
				return t.Fingerprint()
			},
			expectFunc: func(t *MockTask) {
				t.EXPECT().Fingerprint().Return("fp1")
			},
			expected: "fp1",
		},
		{
			tcase: "BlockTTL func",
			task:  NewMockTask(ctrl),
			taskFunc: func(t Task) interface{} {
				return t.BlockTTL()
			},
			expectFunc: func(t *MockTask) {
				t.EXPECT().BlockTTL().Return(30 * time.Minute)
			},
			expected: 30 * time.Minute,
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(testUnit.task)
		assert.Equal(t, testUnit.expected, testUnit.taskFunc(testUnit.task), testUnit.tcase)
	}
}

// Test mock for coverage
func TestMockTaskExecutorCoverage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	task := NewMockTask(ctrl)

	type testTableData struct {
		tcase        string
		executor     *MockTaskExecutor
		params       []interface{}
		executorFunc func(t TaskExecutor, params []interface{}) interface{}
		expectFunc   func(t *MockTaskExecutor, params []interface{})
		expected     interface{}
	}

	testTable := []testTableData{
		{
			tcase:    "NewTask func",
			executor: NewMockTaskExecutor(ctrl),
			params: []interface{}{
				"testevent1", "testrules1", "testalert1", 10 * time.Minute, map[string]interface{}{},
			},
			executorFunc: func(t TaskExecutor, params []interface{}) interface{} {
				return t.NewTask(params[0].(string), params[1].(string), params[2].(string), params[3].(time.Duration), params[4].(map[string]interface{}))
			},
			expectFunc: func(t *MockTaskExecutor, params []interface{}) {
				t.EXPECT().NewTask(params[0].(string), params[1].(string), params[2].(string), params[3].(time.Duration), params[4].(map[string]interface{})).Return(task)
			},
			expected: task,
		},
		{
			tcase:    "ValidateParameters func",
			executor: NewMockTaskExecutor(ctrl),
			params: []interface{}{
				map[string]interface{}{"some param": "some param value"},
			},
			executorFunc: func(t TaskExecutor, params []interface{}) interface{} {
				return t.ValidateParameters(params[0].(map[string]interface{}))
			},
			expectFunc: func(t *MockTaskExecutor, params []interface{}) {
				t.EXPECT().ValidateParameters(params[0].(map[string]interface{})).Return(errors.New("validate error"))
			},
			expected: errors.New("validate error"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(testUnit.executor, testUnit.params)
		assert.Equal(t, testUnit.expected, testUnit.executorFunc(testUnit.executor, testUnit.params), testUnit.tcase)
	}
}

func TestTaskBase_ExecutorInterface(t *testing.T) {
	t.Parallel()

	task := &TaskBase{
		eventID:  "825e",
		rule:     "testrule1",
		alert:    "testalert1",
		blockTTL: 1 * time.Second,
	}

	type testTableData struct {
		tcase    string
		taskFunc func(t *TaskBase) interface{}
		expected interface{}
	}

	testTable := []testTableData{
		{
			tcase: "EventID func",
			taskFunc: func(t *TaskBase) interface{} {
				return t.EventID()
			},
			expected: "825e",
		},
		{
			tcase: "Rule func",
			taskFunc: func(t *TaskBase) interface{} {
				return t.Rule()
			},
			expected: "testrule1",
		},
		{
			tcase: "Alert func",
			taskFunc: func(t *TaskBase) interface{} {
				return t.Alert()
			},
			expected: "testalert1",
		},
		{
			tcase: "BlockTTL func",
			taskFunc: func(t *TaskBase) interface{} {
				return t.BlockTTL()
			},
			expected: 1 * time.Second,
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.taskFunc(task), testUnit.tcase)
	}
}

func TestTaskBase_SetBase(t *testing.T) {
	t.Parallel()

	task := &TaskBase{}
	task.SetBase("825e", "testrule1", "testalert1", 1*time.Second)

	expected := &TaskBase{
		eventID:  "825e",
		rule:     "testrule1",
		alert:    "testalert1",
		blockTTL: 1 * time.Second,
	}

	assert.Equal(t, expected, task)
}
