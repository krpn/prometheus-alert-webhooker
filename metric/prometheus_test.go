package metric

import (
	"errors"
	"github.com/golang/mock/gomock"
	pr "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPrometheus_ConsistentLabelCardinality(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	p := New()

	p.IncomeTaskInc("testrule1", "testalert1", "testexecutor1")
	p.ExecutedTaskObserve("testrule1", "testalert1", "testexecutor1", "success", nil, time.Second)
}

func TestPrometheusm_IncomeTaskInc(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	incomeTasks := NewMockincomeTasks(ctrl)
	prometheus := &PrometheusMetrics{incomeTasks: incomeTasks}

	type testTableData struct {
		tcase                 string
		rule, alert, executor string
		expectFunc            func(m *MockincomeTasks, rule, alert, executor string)
	}

	testTable := []testTableData{
		{
			tcase:    "metric inc",
			rule:     "testrule1",
			alert:    "testalert1",
			executor: "testexecutor1",
			expectFunc: func(m *MockincomeTasks, rule, alert, executor string) {
				m.EXPECT().WithLabelValues(rule, alert, executor).Return(pr.NewCounter(pr.CounterOpts{}))
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(incomeTasks, testUnit.rule, testUnit.alert, testUnit.executor)
		prometheus.IncomeTaskInc(testUnit.rule, testUnit.alert, testUnit.executor)
	}
}

func TestPrometheusm_ExecutedTaskObserve(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executedTasks := NewMockexcutedTasks(ctrl)
	prometheus := &PrometheusMetrics{excutedTasks: executedTasks}

	type testTableData struct {
		tcase                         string
		rule, alert, executor, result string
		err                           error
		duration                      time.Duration
		expectFunc                    func(m *MockexcutedTasks, rule, alert, executor, result string, err error, duration time.Duration)
	}

	testTable := []testTableData{
		{
			tcase:    "metric inc without error",
			rule:     "testrule1",
			alert:    "testalert1",
			executor: "testexecutor1",
			result:   "success",
			err:      nil,
			duration: time.Second,
			expectFunc: func(m *MockexcutedTasks, rule, alert, executor, result string, err error, duration time.Duration) {
				var errText string
				if err != nil {
					errText = err.Error()
				}
				m.EXPECT().WithLabelValues(rule, alert, executor, result, errText).Return(pr.NewHistogram(pr.HistogramOpts{}))
			},
		},
		{
			tcase:    "metric inc with error",
			rule:     "testrule1",
			alert:    "testalert1",
			executor: "testexecutor1",
			result:   "success",
			err:      errors.New("exec error"),
			duration: time.Second,
			expectFunc: func(m *MockexcutedTasks, rule, alert, executor, result string, err error, duration time.Duration) {
				m.EXPECT().WithLabelValues(rule, alert, executor, result, errTextOrEmpty(err)).Return(pr.NewHistogram(pr.HistogramOpts{}))
			},
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(executedTasks, testUnit.rule, testUnit.alert, testUnit.executor, testUnit.result, testUnit.err, testUnit.duration)
		prometheus.ExecutedTaskObserve(testUnit.rule, testUnit.alert, testUnit.executor, testUnit.result, testUnit.err, testUnit.duration)
	}
}

func TestErrTextOrEmpty(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		err      error
		expected string
	}

	testTable := []testTableData{
		{
			tcase:    "error is not nil",
			err:      errors.New("some error"),
			expected: "some error",
		},
		{
			tcase:    "error is nil",
			err:      nil,
			expected: "",
		},
	}

	for _, testUnit := range testTable {
		result := errTextOrEmpty(testUnit.err)
		assert.Equal(t, testUnit.expected, result, testUnit.tcase)
	}
}
