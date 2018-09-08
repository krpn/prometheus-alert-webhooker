package model

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func TestRule_validateUncompiled(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		rule     func() Rule
		expected error
	}

	testTable := []testTableData{
		{
			tcase:    "valid",
			rule:     func() Rule { return *getTestRuleUncompiled(1) },
			expected: nil,
		},
		{
			tcase: "empty rule name",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Name = ""
				return rule
			},
			expected: errRuleValidateEmptyName,
		},
		{
			tcase: "already compiled labels",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertLabelsRegexp: map[string]*regexp.Regexp{
						"a": regexp.MustCompile("b(.*?)"),
					},
				}
				return rule
			},
			expected: errRuleValidateAlreadyCompiled,
		},
		{
			tcase: "already compiled annotations",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertAnnotationsRegexp: map[string]*regexp.Regexp{
						"a": regexp.MustCompile("b(.*?)"),
					},
				}
				return rule
			},
			expected: errRuleValidateAlreadyCompiled,
		},
		{
			tcase: "invalid status",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertStatus: "test",
					AlertLabels: map[string]string{
						"a": "b",
					},
				}
				return rule
			},
			expected: errRuleValidateInvalidAlertStatus,
		},
		{
			tcase: "empty alert label name",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertStatus: "firing",
					AlertLabels: map[string]string{
						"": "b",
					},
				}
				return rule
			},
			expected: errors.New("alert label validation error: key is empty"),
		},
		{
			tcase: "empty alert label value",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertStatus: "firing",
					AlertLabels: map[string]string{
						"a": "",
					},
				}
				return rule
			},
			expected: errors.New("alert label validation error: value for key a is empty"),
		},
		{
			tcase: "empty annotation label name",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertStatus: "firing",
					AlertAnnotations: map[string]string{
						"": "b",
					},
				}
				return rule
			},
			expected: errors.New("alert annotation validation error: key is empty"),
		},
		{
			tcase: "empty annotation label value",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions = Conditions{
					AlertStatus: "firing",
					AlertAnnotations: map[string]string{
						"a": "",
					},
				}
				return rule
			},
			expected: errors.New("alert annotation validation error: value for key a is empty"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.rule().validateUncompiled(), testUnit.tcase)
	}
}

func Test_validateAlertStatus(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		status   string
		expected error
	}

	testTable := []testTableData{
		{
			tcase:    "firing",
			status:   "firing",
			expected: nil,
		},
		{
			tcase:    "resolved",
			status:   "resolved",
			expected: nil,
		},
		{
			tcase:    "empty",
			status:   "",
			expected: nil,
		},
		{
			tcase:    "invalid",
			status:   "invalid",
			expected: errRuleValidateInvalidAlertStatus,
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, validateAlertStatus(testUnit.status), testUnit.tcase)
	}
}

func TestRule_setDefaultAlertStatus(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		rule     func() Rule
		expected Rule
	}

	testTable := []testTableData{
		{
			tcase:    "firing to default",
			rule:     func() Rule { return *getTestRuleUncompiled(1) },
			expected: *getTestRuleUncompiled(1),
		},
		{
			tcase: "resolved to default",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertStatus = string(model.AlertResolved)
				return rule
			},
			expected: *getTestRuleUncompiled(1),
		},
		{
			tcase: "empty to default",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertStatus = ""
				return rule
			},
			expected: *getTestRuleUncompiled(1),
		},
	}

	for _, testUnit := range testTable {
		rule := testUnit.rule()
		rule.setDefaultAlertStatus()
		assert.Equal(t, testUnit.expected, rule, testUnit.tcase)
	}
}

func TestRule_compile(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		rule     func() Rule
		expected func() Rule
	}

	testTable := []testTableData{
		{
			tcase:    "compile one",
			rule:     func() Rule { return *getTestRuleUncompiled(1) },
			expected: func() Rule { return *getTestRuleCompiled(1) },
		},
		{
			tcase: "compile one of two",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertLabels = map[string]string{
					"a": "b",
					"c": "^d(.*?)",
				}
				rule.Conditions.AlertAnnotations = map[string]string{}
				return rule
			},
			expected: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertLabels = map[string]string{
					"a": "b",
				}
				rule.Conditions.AlertLabelsRegexp = map[string]*regexp.Regexp{
					"c": regexp.MustCompile("^d(.*?)"),
				}
				rule.Conditions.AlertAnnotations = map[string]string{}
				return rule
			},
		},
		{
			tcase: "can not compile one of two ",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertLabels = map[string]string{
					"a": "b:)",
					"c": "^d(.*?)",
				}
				rule.Conditions.AlertAnnotations = map[string]string{}
				return rule
			},
			expected: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertLabels = map[string]string{
					"a": "b:)",
				}
				rule.Conditions.AlertLabelsRegexp = map[string]*regexp.Regexp{
					"c": regexp.MustCompile("^d(.*?)"),
				}
				rule.Conditions.AlertAnnotations = map[string]string{}
				return rule
			},
		},
		{
			tcase: "compile one only annotations",
			rule: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertLabels = map[string]string{}
				return rule
			},
			expected: func() Rule {
				rule := *getTestRuleUncompiled(1)
				rule.Conditions.AlertLabels = map[string]string{}
				rule.Conditions.AlertLabelsRegexp = map[string]*regexp.Regexp{}
				rule.Conditions.AlertAnnotations = map[string]string{}
				rule.Conditions.AlertAnnotationsRegexp = map[string]*regexp.Regexp{
					"aa": regexp.MustCompile("ab(.*?)"),
				}
				return rule
			},
		},
	}

	for _, testUnit := range testTable {
		rule := testUnit.rule()
		rule.compile()
		assert.Equal(t, testUnit.expected(), rule, testUnit.tcase)
	}
}

func TestRules_Prepare(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executorMock := executor.NewMockTaskExecutor(ctrl)

	type testTableData struct {
		tcase            string
		rules            func() Rules
		commonParameters map[string]map[string]interface{}
		executors        map[string]executor.TaskExecutor
		expectFunc       func(e *executor.MockTaskExecutor)
		expectedRules    func() Rules
		expectedErr      error
	}

	testTable := []testTableData{
		{
			tcase: "no errors",
			rules: func() Rules { return Rules{*getTestRuleUncompiled(1)} },
			executors: map[string]executor.TaskExecutor{
				"shell": executorMock,
			},
			expectFunc: func(e *executor.MockTaskExecutor) {
				e.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(nil)
			},
			expectedRules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Conditions.AlertLabels = map[string]string{
						"a": "b",
					}
					rule.Conditions.AlertLabelsRegexp = map[string]*regexp.Regexp{}
					rule.Conditions.AlertAnnotations = map[string]string{}
					rule.Conditions.AlertAnnotationsRegexp = map[string]*regexp.Regexp{
						"aa": regexp.MustCompile("ab(.*?)"),
					}
					rule.Actions = Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
							},
							Block:        10 * time.Second,
							TaskExecutor: executorMock,
						},
					}

					r[i] = rule
				}
				return r
			},
			expectedErr: nil,
		},
		{
			tcase: "validate uncompiled error",
			rules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Conditions.AlertLabels = map[string]string{
						"": "empty_label",
					}
					r[i] = rule
				}
				return r
			},
			executors: map[string]executor.TaskExecutor{
				"shell": executorMock,
			},
			expectFunc: func(e *executor.MockTaskExecutor) {},
			expectedRules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Conditions.AlertLabels = map[string]string{
						"": "empty_label",
					}
					r[i] = rule
				}
				return r
			},
			expectedErr: errors.New("alert label validation error: key is empty"),
		},
		{
			tcase: "empty alert status",
			rules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Conditions.AlertStatus = ""
					r[i] = rule
				}
				return r
			},
			executors: map[string]executor.TaskExecutor{
				"shell": executorMock,
			},
			expectFunc: func(e *executor.MockTaskExecutor) {
				e.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(nil)
			},
			expectedRules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Conditions.AlertLabels = map[string]string{
						"a": "b",
					}
					rule.Conditions.AlertLabelsRegexp = map[string]*regexp.Regexp{}
					rule.Conditions.AlertAnnotations = map[string]string{}
					rule.Conditions.AlertAnnotationsRegexp = map[string]*regexp.Regexp{
						"aa": regexp.MustCompile("ab(.*?)"),
					}
					rule.Actions = Actions{
						{
							Executor: "shell",
							Parameters: map[string]interface{}{
								"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
							},
							Block:        10 * time.Second,
							TaskExecutor: executorMock,
						},
					}

					r[i] = rule
				}
				return r
			},
			expectedErr: nil,
		},
		{
			tcase: "empty actions error",
			rules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Actions = nil
					r[i] = rule
				}
				return r
			},
			executors: map[string]executor.TaskExecutor{
				"shell": executorMock,
			},
			expectFunc: func(e *executor.MockTaskExecutor) {},
			expectedRules: func() Rules {
				rules := Rules{*getTestRuleUncompiled(1)}
				r := make(Rules, len(rules))
				for i, rule := range rules {
					rule.Actions = nil
					r[i] = rule
				}
				return r
			},
			expectedErr: errRuleValidateEmptyActions,
		},
		{
			tcase:         "zero executors error",
			rules:         func() Rules { return Rules{*getTestRuleUncompiled(1)} },
			executors:     nil,
			expectFunc:    func(e *executor.MockTaskExecutor) {},
			expectedRules: func() Rules { return Rules{*getTestRuleUncompiled(1)} },
			expectedErr:   errRuleValidateEmptyExecutors,
		},

		{
			tcase: "zero executors error",
			rules: func() Rules { return Rules{} },
			executors: map[string]executor.TaskExecutor{
				"shell": executorMock,
			},
			expectFunc:    func(e *executor.MockTaskExecutor) {},
			expectedRules: func() Rules { return Rules{} },
			expectedErr:   errRulesValidateEmptyRules,
		},
		{
			tcase: "zero executors error",
			rules: func() Rules { return nil },
			executors: map[string]executor.TaskExecutor{
				"shell": executorMock,
			},
			expectFunc:    func(e *executor.MockTaskExecutor) {},
			expectedRules: func() Rules { return nil },
			expectedErr:   errRulesValidateEmptyRules,
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(executorMock)
		rules := testUnit.rules()
		err := rules.Prepare(testUnit.commonParameters, testUnit.executors)
		assert.Equal(t, testUnit.expectedRules(), rules, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}
}

func TestRule_mergeCommonParameters(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase            string
		rule             func() *Rule
		commonParameters map[string]map[string]interface{}
		expected         func() *Rule
	}

	testTable := []testTableData{
		{
			tcase: "merged",
			rule: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "shell",
						CommonParameters: "jenkins",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			commonParameters: map[string]map[string]interface{}{
				"jenkins": {
					"login":    "admin",
					"password": "abc",
				},
			},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "shell",
						CommonParameters: "jenkins",
						Parameters: map[string]interface{}{
							"command":  "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
							"login":    "admin",
							"password": "abc",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
		},
		{
			tcase: "common parameters has low priority to parameters",
			rule: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "shell",
						CommonParameters: "jenkins",
						Parameters: map[string]interface{}{
							"password": "abcfromparams",
							"command":  "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			commonParameters: map[string]map[string]interface{}{
				"jenkins": {
					"login":    "admin",
					"password": "abc",
				},
			},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "shell",
						CommonParameters: "jenkins",
						Parameters: map[string]interface{}{
							"command":  "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
							"login":    "admin",
							"password": "abcfromparams",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
		},
		{
			tcase: "common parameters not found",
			rule: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "shell",
						CommonParameters: "gitlab",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			commonParameters: map[string]map[string]interface{}{
				"jenkins": {
					"login":    "admin",
					"password": "abc",
				},
			},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "shell",
						CommonParameters: "gitlab",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
		},
		{
			tcase: "empty common parameters setting in action",
			rule:  func() *Rule { return getTestRuleUncompiled(1) },
			commonParameters: map[string]map[string]interface{}{
				"jenkins": {
					"login":    "admin",
					"password": "abc",
				},
			},
			expected: func() *Rule { return getTestRuleUncompiled(1) },
		},
		{
			tcase:            "empty common parameters list",
			rule:             func() *Rule { return getTestRuleUncompiled(1) },
			commonParameters: nil,
			expected:         func() *Rule { return getTestRuleUncompiled(1) },
		},
		{
			tcase: "parameters only from common parameters",
			rule: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "telegram",
						CommonParameters: "telegram_bot",
						Parameters:       nil,
						Block:            10 * time.Second,
					},
				}
				return rule
			},
			commonParameters: map[string]map[string]interface{}{
				"telegram_bot": {
					"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
					"chat_id":   12345678,
					"message":   "test",
				},
			},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor:         "telegram",
						CommonParameters: "telegram_bot",
						Parameters: map[string]interface{}{
							"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
							"chat_id":   12345678,
							"message":   "test",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
		},
	}

	for _, testUnit := range testTable {
		rule := testUnit.rule()
		rule.mergeCommonParameters(testUnit.commonParameters)
		assert.Equal(t, testUnit.expected(), rule, testUnit.tcase)
	}
}

func TestRule_prepareTaskExecutors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executorMock := executor.NewMockTaskExecutor(ctrl)

	type testTableData struct {
		tcase         string
		rule          func() *Rule
		taskExecutors map[string]executor.TaskExecutor
		expectFunc    func(e *executor.MockTaskExecutor)
		expected      func() *Rule
		expectedErr   error
	}

	testTable := []testTableData{
		{
			tcase:         "success",
			rule:          func() *Rule { return getTestRuleUncompiled(1) },
			taskExecutors: map[string]executor.TaskExecutor{"shell": executorMock},
			expectFunc: func(e *executor.MockTaskExecutor) {
				e.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(nil)
			},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor: "shell",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block:        10 * time.Second,
						TaskExecutor: executorMock,
					},
				}
				return rule
			},
			expectedErr: nil,
		},
		{
			tcase:         "validate params error",
			rule:          func() *Rule { return getTestRuleUncompiled(1) },
			taskExecutors: map[string]executor.TaskExecutor{"shell": executorMock},
			expectFunc: func(e *executor.MockTaskExecutor) {
				e.EXPECT().ValidateParameters(map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				}).Return(errors.New("validate params error"))
			},
			expected:    func() *Rule { return getTestRuleUncompiled(1) },
			expectedErr: errors.New("validate params error"),
		},
		{
			tcase:         "empty executors",
			rule:          func() *Rule { return getTestRuleUncompiled(1) },
			taskExecutors: nil,
			expectFunc:    func(e *executor.MockTaskExecutor) {},
			expected:      func() *Rule { return getTestRuleUncompiled(1) },
			expectedErr:   errRuleValidateEmptyExecutors,
		},
		{
			tcase: "empty actions",
			rule: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			taskExecutors: map[string]executor.TaskExecutor{"shell": executorMock},
			expectFunc:    func(e *executor.MockTaskExecutor) {},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			expectedErr: errRuleValidateEmptyExecutor,
		},
		{
			tcase: "empty actions",
			rule: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor: "jenkins",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			taskExecutors: map[string]executor.TaskExecutor{"shell": executorMock},
			expectFunc:    func(e *executor.MockTaskExecutor) {},
			expected: func() *Rule {
				rule := getTestRuleUncompiled(1)
				rule.Actions = Actions{
					{
						Executor: "jenkins",
						Parameters: map[string]interface{}{
							"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
						},
						Block: 10 * time.Second,
					},
				}
				return rule
			},
			expectedErr: errors.New("executor jenkins not found"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(executorMock)
		rule := testUnit.rule()
		err := rule.prepareTaskExecutors(testUnit.taskExecutors)
		assert.Equal(t, testUnit.expected(), rule, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}
}

func getTestRuleUncompiled(num int) *Rule {
	return &Rule{
		Name: fmt.Sprintf("testrule%v", num),
		Conditions: Conditions{
			AlertStatus: "firing",
			AlertLabels: map[string]string{
				"a": "b",
			},
			AlertLabelsRegexp: map[string]*regexp.Regexp{},
			AlertAnnotations: map[string]string{
				"aa": "ab(.*?)",
			},
			AlertAnnotationsRegexp: map[string]*regexp.Regexp{},
		},
		Actions: Actions{
			{
				Executor: "shell",
				Parameters: map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				},
				Block: 10 * time.Second,
			},
		},
	}
}

func getTestRuleCompiled(num int) *Rule {
	return &Rule{
		Name: fmt.Sprintf("testrule%v", num),
		Conditions: Conditions{
			AlertStatus: "firing",
			AlertLabels: map[string]string{
				"a": "b",
			},
			AlertLabelsRegexp: map[string]*regexp.Regexp{},
			AlertAnnotations:  map[string]string{},
			AlertAnnotationsRegexp: map[string]*regexp.Regexp{
				"aa": regexp.MustCompile("ab(.*?)"),
			},
		},
		Actions: Actions{
			{
				Executor: "shell",
				Parameters: map[string]interface{}{
					"command": "${LABEL_BLOCK} | ${URLENCODE_LABEL_ERROR} | ${CUT_AFTER_LAST_COLON_LABEL_INSTANCE} | ${ANNOTATION_TITLE}",
				},
				Block: 10 * time.Second,
			},
		},
	}
}
