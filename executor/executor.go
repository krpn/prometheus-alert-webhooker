package executor

import (
	"github.com/sirupsen/logrus"
	"time"
)

//go:generate mockgen -source=executor.go -destination=executor_mocks.go -package=executor doc github.com/golang/mock/gomock

// Task is the interface implemented by executor
// used for execute tasks and get task information.
type Task interface {
	// EventID returns event ID.
	EventID() string

	// Rule returns rule name.
	Rule() string

	// Alert returns alert name.
	Alert() string

	// BlockTTL returns TTL for blocking task after execute.
	// If return 0, task not blocked.
	BlockTTL() time.Duration

	ExecutorName() string

	// ExecutorDetails returns structured information about task (for example, parameters)
	// It used for TaskDetails function.
	ExecutorDetails() interface{}

	// Fingerprint returns uniq string represents task fingerprint.
	// It used for blocking task.
	Fingerprint() string

	// Exec executes task.
	Exec(logger *logrus.Logger) error
}

// TaskExecutor is the interface implemented by executor
// used for validate task parameters and create tasks.
type TaskExecutor interface {
	NewTask(eventID, rule, alert string, blockTTL time.Duration, preparedParameters map[string]interface{}) Task
	ValidateParameters(parameters map[string]interface{}) error
}

// TaskDetails returns task details as a map.
// It used for logging.
func TaskDetails(task Task) map[string]interface{} {
	return map[string]interface{}{
		"event_id": task.EventID(),
		"rule":     task.Rule(),
		"alert":    task.Alert(),
		"executor": task.ExecutorName(),
		"details":  task.ExecutorDetails(),
	}
}

// TaskBase implements basic Task methods.
type TaskBase struct {
	eventID  string
	rule     string
	alert    string
	blockTTL time.Duration
}

// EventID implements basic Task methods, returns private field.
func (task *TaskBase) EventID() string {
	return task.eventID
}

// Rule implements basic Task methods, returns private field.
func (task *TaskBase) Rule() string {
	return task.rule
}

// Alert implements basic Task methods, returns private field.
func (task *TaskBase) Alert() string {
	return task.alert
}

// BlockTTL implements basic Task methods, returns private field.
func (task *TaskBase) BlockTTL() time.Duration {
	return task.blockTTL
}

// SetBase sets basic private fields.
func (task *TaskBase) SetBase(eventID, rule, alert string, blockTTL time.Duration) {
	task.eventID = eventID
	task.rule = rule
	task.alert = alert
	task.blockTTL = blockTTL
}
