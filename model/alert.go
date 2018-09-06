package model

import (
	"github.com/krpn/prometheus-alert-webhooker/utils"
	"github.com/prometheus/common/model"
	"regexp"
)

type alert struct {
	Status      string
	Labels      map[string]string
	Annotations map[string]string
}

func (a alert) match(conditions Conditions) bool {
	if a.Status != conditions.AlertStatus {
		return false
	}

	match := mapMatchConditions(a.Labels, conditions.AlertLabels, conditions.AlertLabelsRegexp)
	if !match {
		return false
	}

	return mapMatchConditions(a.Annotations, conditions.AlertAnnotations, conditions.AlertAnnotationsRegexp)
}

func mapMatchConditions(m map[string]string, conditions map[string]string, conditionsR map[string]*regexp.Regexp) bool {
	for label, value := range conditions {
		avalue, ok := m[label]
		if !ok {
			return false
		}

		if avalue != value {
			return false
		}
	}

	for label, rvalue := range conditionsR {
		avalue, ok := m[label]
		if !ok {
			return false
		}

		if !rvalue.MatchString(avalue) {
			return false
		}
	}

	return true
}

func (a alert) toTasksGroups(rules Rules, eventID string) (tasksGroups TasksGroups) {
	tasksGroups = make(TasksGroups, 0)

	for _, rule := range rules {
		if !a.match(rule.Conditions) {
			continue
		}

		tasksGroups = append(tasksGroups, NewTasks(rule, a, eventID))
	}

	return
}

func (a alert) Name() string {
	return a.Labels[model.AlertNameLabel]
}

// Alerts  is a slice of Alert.
type Alerts []alert

// ToTasksGroups converts alerts to tasks.
func (alerts Alerts) ToTasksGroups(rules Rules, eventID string) (tasksGroups TasksGroups) {
	tasksGroups = make(TasksGroups, 0)

	for _, alert := range alerts {
		tasksGroups = append(tasksGroups, alert.toTasksGroups(rules, eventID)...)
	}

	return
}

func prepareParams(params map[string]interface{}, alert alert) map[string]interface{} {
	preparedParams := make(map[string]interface{}, len(params))

	for param, value := range params {
		valueStr, ok := value.(string)
		if !ok {
			preparedParams[param] = value
			continue
		}

		for annotation, value := range alert.Annotations {
			valueStr = utils.ReplacePlaceholders(valueStr, "ANNOTATION", annotation, value)
		}

		for label, value := range alert.Labels {
			valueStr = utils.ReplacePlaceholders(valueStr, "LABEL", label, value)
		}

		preparedParams[param] = valueStr
	}

	return preparedParams
}
