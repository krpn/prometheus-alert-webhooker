package shell

import (
	"errors"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/sirupsen/logrus"
	"os/exec"
	"time"
)

const (
	paramCommand = "command"
	paramArgs    = "args"
)

type task struct {
	executor.TaskBase
	execFunc func(name string, arg ...string) *exec.Cmd
	command  string
	args     []string
}

func (task *task) ExecutorName() string {
	return "shell"
}

func (task *task) ExecutorDetails() interface{} {
	return map[string]interface{}{"command": task.command}
}

func (task *task) Fingerprint() string {
	return utils.MD5Hash(task.command)
}

func (task *task) Exec(logger *logrus.Logger) error {
	cmd := task.execFunc(task.command, task.args...)
	_, err := cmd.Output()
	return err
}

type taskExecutor struct {
	execFunc func(name string, arg ...string) *exec.Cmd
}

// NewExecutor creates TaskExecutor for shell tasks.
func NewExecutor(execFunc func(string, ...string) *exec.Cmd) executor.TaskExecutor {
	return taskExecutor{execFunc: execFunc}
}

func (executor taskExecutor) ValidateParameters(parameters map[string]interface{}) error {
	command, ok := parameters[paramCommand]
	if !ok {
		return errors.New("required parameter command is missing")
	}

	_, ok = command.(string)
	if !ok {
		return errors.New("command parameter value is not a string")
	}

	return nil
}

func (executor taskExecutor) NewTask(eventID, rule, alert string, blockTTL time.Duration, preparedParameters map[string]interface{}) executor.Task {

	var args []string
	if _, ok := preparedParameters[paramArgs]; ok {
		argsIface, _ := preparedParameters[paramArgs].([]interface{})
		args = make([]string, len(argsIface))
		for i := range argsIface {
			args[i] = argsIface[i].(string)
		}
	}
	task := &task{
		execFunc: executor.execFunc,
		command:  preparedParameters[paramCommand].(string),
		args:     args,
	}
	task.SetBase(eventID, rule, alert, blockTTL)
	return task
}
