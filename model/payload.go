package model

import "github.com/prometheus/alertmanager/template"

// Payload represents json structure of payload from Alertmanager.
type Payload template.Data

// ToAlerts converts payload to alerts.
func (payload Payload) ToAlerts() (alerts Alerts) {
	alerts = make(Alerts, len(payload.Alerts))

	for i, a := range payload.Alerts {

		labels := make(map[string]string, len(payload.CommonLabels)+len(a.Labels))
		for key, val := range payload.CommonLabels {
			labels[key] = val
		}
		for key, val := range a.Labels {
			labels[key] = val
		}

		annotations := make(map[string]string, len(payload.CommonAnnotations)+len(a.Annotations))
		for key, val := range payload.CommonAnnotations {
			annotations[key] = val
		}
		for key, val := range a.Annotations {
			annotations[key] = val
		}

		alerts[i] = alert{
			Status:      payload.Status,
			Labels:      labels,
			Annotations: annotations,
		}
	}

	return
}
