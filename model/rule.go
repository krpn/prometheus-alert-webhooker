package model

import (
	"errors"
	"fmt"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/prometheus/common/model"
	"regexp"
	"strings"
)

// Rule describes rule for alerts.
type Rule struct {
	// Name of the rule, used for metrics, logger.
	Name string `mapstructure:"name"`

	// Conditions for rule match.
	Conditions Conditions `mapstructure:"conditions"`

	// Actions is a slice of action.
	Actions Actions `mapstructure:"actions"`
}

// Conditions describes
type Conditions struct {
	// AlertStatus is a status of alert. By default set by setDefaultAlertStatus() function.
	AlertStatus string `mapstructure:"alert_status"`

	// AlertLabels is a map label-value for match labels.
	AlertLabels map[string]string `mapstructure:"alert_labels"`

	// AlertLabelsRegexp is a compiled AlertLabels.
	AlertLabelsRegexp map[string]*regexp.Regexp `mapstructure:"-"`

	// AlertAnnotations is a map annotation-value for match annotations.
	AlertAnnotations map[string]string `mapstructure:"alert_annotations"`

	// AlertAnnotationsRegexp is a compiled AlertAnnotations.
	AlertAnnotationsRegexp map[string]*regexp.Regexp `mapstructure:"-"`
}

// Rules is a slice of Rule.
type Rules []Rule

var (
	errRulesValidateEmptyRules               = errors.New("empty rules list")
	errRuleValidateEmptyName                 = errors.New("empty rule name")
	errRuleValidateInvalidAlertStatus        = errors.New("invalid alert status: should be firing or resolved")
	errRuleValidateEmptyAlertLabelName       = errors.New("empty alert label name")
	errRuleValidateEmptyAlertLabelValue      = errors.New("empty alert label value")
	errRuleValidateEmptyAnnotationLabelName  = errors.New("empty alert annotation name")
	errRuleValidateEmptyAnnotationLabelValue = errors.New("empty alert annotation value")
	errRuleValidateEmptyExecutors            = errors.New("empty executors")
	errRuleValidateEmptyActions              = errors.New("empty actions")
	errRuleValidateAlreadyCompiled           = errors.New("rules already compiled")
)

func (rule Rule) validateUncompiled() error {
	if len(rule.Actions) == 0 {
		return errRuleValidateEmptyActions
	}

	if len(rule.Name) == 0 {
		return errRuleValidateEmptyName
	}

	err := validateAlertStatus(rule.Conditions.AlertStatus)
	if err != nil {
		return err
	}

	if len(rule.Conditions.AlertLabelsRegexp) > 0 || len(rule.Conditions.AlertAnnotationsRegexp) > 0 {
		return errRuleValidateAlreadyCompiled
	}

	for label, value := range rule.Conditions.AlertLabels {
		if len(label) == 0 {
			return errRuleValidateEmptyAlertLabelName
		}

		if len(value) == 0 {
			return errRuleValidateEmptyAlertLabelValue
		}
	}

	for annotation, value := range rule.Conditions.AlertAnnotations {
		if len(annotation) == 0 {
			return errRuleValidateEmptyAnnotationLabelName
		}

		if len(value) == 0 {
			return errRuleValidateEmptyAnnotationLabelValue
		}
	}

	return nil
}

func validateAlertStatus(status string) error {
	if len(status) == 0 {
		return nil
	}

	if status == string(model.AlertFiring) {
		return nil
	}

	if status == string(model.AlertResolved) {
		return nil
	}

	return errRuleValidateInvalidAlertStatus
}

func (rule *Rule) setDefaultAlertStatus() {
	rule.Conditions.AlertStatus = string(model.AlertFiring)
}

func (rule *Rule) compile() {
	l, rl := compileMap(rule.Conditions.AlertLabels)
	rule.Conditions.AlertLabels = l
	rule.Conditions.AlertLabelsRegexp = rl

	l, rl = compileMap(rule.Conditions.AlertAnnotations)
	rule.Conditions.AlertAnnotations = l
	rule.Conditions.AlertAnnotationsRegexp = rl
}

func compileMap(m map[string]string) (map[string]string, map[string]*regexp.Regexp) {
	l := make(map[string]string)
	rl := make(map[string]*regexp.Regexp)
	for key, val := range m {
		r, err := regexp.Compile(val)
		if err != nil {
			l[key] = val
			continue
		}

		if r.NumSubexp() == 0 {
			l[key] = val
			continue
		}

		rl[key] = r
	}
	return l, rl
}

// Prepare prepares rules after config init.
func (rules Rules) Prepare(commonParams map[string]map[string]interface{}, taskExecutors map[string]executor.TaskExecutor) error {
	if len(rules) == 0 {
		return errRulesValidateEmptyRules
	}

	var err error
	for i, rule := range rules {
		// validate
		err = rule.validateUncompiled()
		if err != nil {
			return err
		}

		// set default alert status if needed
		if len(rule.Conditions.AlertStatus) == 0 {
			rule.setDefaultAlertStatus()
		}

		rule.mergeCommonParameters(commonParams)

		err = rule.prepareTaskExecutors(taskExecutors)
		if err != nil {
			return err
		}

		// compile regexp
		rule.compile()

		rules[i] = rule
	}

	return nil
}

func (rule *Rule) mergeCommonParameters(commonParams map[string]map[string]interface{}) {
	if len(commonParams) == 0 {
		return
	}

	for _, action := range rule.Actions {
		if action.CommonParameters == "" {
			continue
		}

		common, ok := commonParams[action.CommonParameters]
		if !ok {
			continue
		}

		for param, value := range common {
			_, ok = action.Parameters[param]
			if !ok {
				action.Parameters[param] = value
			}
		}
	}
}

func (rule *Rule) prepareTaskExecutors(taskExecutors map[string]executor.TaskExecutor) error {

	if len(taskExecutors) == 0 {
		return errRuleValidateEmptyExecutors
	}

	for i, action := range rule.Actions {
		TaskExecutor, ok := taskExecutors[strings.ToLower(action.Executor)]
		if !ok {
			return fmt.Errorf("executor for action type %v not found", action.Executor)
		}

		err := TaskExecutor.ValidateParameters(action.Parameters)
		if err != nil {
			return err
		}

		action.TaskExecutor = TaskExecutor
		rule.Actions[i] = action
	}

	return nil
}
