package shell

import (
	"bytes"
	"errors"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
	"time"
)

func TestShellTask_ExecutorInterface(t *testing.T) {
	t.Parallel()

	logger, hook := test.NewNullLogger()

	execFunc := func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{Stdout: &bytes.Buffer{}}
	}

	executorMock := NewExecutor(execFunc)
	task := executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second, map[string]interface{}{"command": "some cmd"})

	type testTableData struct {
		tcase    string
		taskFunc func(t executor.Task) interface{}
		expected interface{}
	}

	testTable := []testTableData{
		{
			tcase: "ExecutorName func",
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorName()
			},
			expected: "shell",
		},
		{
			tcase: "ExecutorDetails func",
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{"command": "some cmd"},
		},
		{
			tcase: "Fingerprint func",
			taskFunc: func(t executor.Task) interface{} {
				return t.Fingerprint()
			},
			expected: "4df3218067010b4c4bd8754ab89c18c2",
		},
		{
			tcase: "Exec func",
			taskFunc: func(t executor.Task) interface{} {
				return t.Exec(logger)
			},
			expected: errors.New("exec: Stdout already set"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.taskFunc(task), testUnit.tcase)
	}

	// logger is not used
	assert.Equal(t, 0, len(hook.Entries))
}

func TestShellTaskExecutor_NewTask(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(nil)

	testTask := executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second, map[string]interface{}{"command": "some cmd"})
	expected := &task{
		execFunc: nil,
		command:  "some cmd",
	}
	expected.SetBase("825e", "testrule1", "testalert1", 1*time.Second)

	assert.Equal(t, expected, testTask)
}

func TestShellTaskExecutor_ValidateParameters(t *testing.T) {
	t.Parallel()

	execFunc := func(name string, arg ...string) *exec.Cmd {
		return &exec.Cmd{Stdout: &bytes.Buffer{}}
	}

	executorMock := NewExecutor(execFunc)

	type testTableData struct {
		tcase    string
		params   map[string]interface{}
		expected error
	}

	testTable := []testTableData{
		{
			tcase:    "correct params",
			params:   map[string]interface{}{"command": "some command"},
			expected: nil,
		},
		{
			tcase:    "param missing",
			params:   map[string]interface{}{"login": "admin"},
			expected: errors.New("required parameter command is missing"),
		},
		{
			tcase:    "param wrong type",
			params:   map[string]interface{}{"command": 123},
			expected: errors.New("command parameter value is not a string"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, executorMock.ValidateParameters(testUnit.params), testUnit.tcase)
	}
}
