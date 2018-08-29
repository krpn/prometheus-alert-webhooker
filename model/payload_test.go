package model

import (
	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPayload_ToAlerts(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		payload  Payload
		expected Alerts
	}

	testTable := []testTableData{
		{
			payload: Payload(template.Data{
				Alerts: []template.Alert{
					{
						Labels: map[string]string{
							"label1": "value1",
						},
						Annotations: map[string]string{
							"annotation1": "avalue1",
						},
					},
					{
						Labels: map[string]string{
							"label2": "value2",
						},
					},
				},
				Status: "firing",
				CommonLabels: map[string]string{
					"clabel1": "cvalue1",
				},
				CommonAnnotations: map[string]string{
					"cannotation1": "cavalue1",
				},
			}),
			expected: []alert{
				{
					Status: "firing",
					Labels: map[string]string{
						"clabel1": "cvalue1",
						"label1":  "value1",
					},
					Annotations: map[string]string{
						"cannotation1": "cavalue1",
						"annotation1":  "avalue1",
					},
				},
				{
					Status: "firing",
					Labels: map[string]string{
						"clabel1": "cvalue1",
						"label2":  "value2",
					},
					Annotations: map[string]string{
						"cannotation1": "cavalue1",
					},
				},
			},
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.payload.ToAlerts(), testUnit.tcase)
	}
}
