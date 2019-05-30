package model

import (
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"time"
)

// Action describes direct action as an reaction of alert.
type Action struct {
	// Executor of action: shell, jenkins, etc.
	Executor string `mapstructure:"executor"`

	// CommonParameters represents string for Config.CommonParameters map.
	CommonParameters string `mapstructure:"common_parameters"`

	// Parameters for TaskExecutor.
	Parameters map[string]interface{} `mapstructure:"parameters"`

	// Block time after action success execute.
	Block time.Duration `mapstructure:"block"`

	// TaskExecutor for this action.
	TaskExecutor executor.TaskExecutor `mapstructure:"-"`
}

// Actions is a slice of Action.
type Actions []Action
