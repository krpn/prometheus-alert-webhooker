package model

import "github.com/krpn/prometheus-alert-webhooker/executor"

// Tasks is a slice of executor.Task.
type Tasks []executor.Task

// Details gets details for all tasks.
func (tasks Tasks) Details() []map[string]interface{} {
	r := make([]map[string]interface{}, len(tasks))

	for i, task := range tasks {
		r[i] = executor.TaskDetails(task)
	}

	return r
}

// NewTasks creates for rule-alert pairs.
func NewTasks(rule Rule, alert alert, eventID string) Tasks {
	tasks := make(Tasks, 0)

	for _, action := range rule.Actions {
		preparedParams := prepareParams(action.Parameters, alert)
		tasks = append(tasks, action.TaskExecutor.NewTask(eventID, rule.Name, alert.Name(), action.Block, preparedParams))
	}

	return tasks
}
