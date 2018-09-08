package jenkins

import (
	"errors"
	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestJenkinsTask_ExecutorInterface(t *testing.T) {
	t.Parallel()

	taskWithParams := NewExecutor().NewTask("id", "rule", "alert", 10*time.Minute, map[string]interface{}{
		"endpoint":                "http://jenkins.company.com/",
		"job":                     "SomeJob",
		"login":                   "admin",
		"password":                "qwerty123",
		"job parameter test":      "test1",
		"job parameter bad_param": 123,
	})
	taskWithoutParams := NewExecutor().NewTask("id", "rule", "alert", 10*time.Minute, map[string]interface{}{
		"endpoint": "http://jenkins.company.com/",
		"job":      "SomeJob",
		"login":    "admin",
		"password": "qwerty123",
	})

	type testTableData struct {
		tcase    string
		task     executor.Task
		taskFunc func(t executor.Task) interface{}
		expected interface{}
	}

	testTable := []testTableData{
		{
			tcase: "ExecutorName func",
			task:  taskWithParams,
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorName()
			},
			expected: "Jenkins",
		},
		{
			tcase: "ExecutorDetails func with params",
			task:  taskWithParams,
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{"job": "SomeJob", "parameters": map[string]string{"test": "test1"}},
		},
		{
			tcase: "ExecutorDetails func without params",
			task:  taskWithoutParams,
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{"job": "SomeJob"},
		},
		{
			tcase: "Fingerprint func",
			task:  taskWithParams,
			taskFunc: func(t executor.Task) interface{} {
				return t.Fingerprint()
			},
			expected: "ffea66a1ae8abf76cc90edda3faccdd3",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.taskFunc(testUnit.task), testUnit.tcase)
	}
}

func TestJenkinsTaskExecutor_ValidateParameters(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor()

	type testTableData struct {
		tcase    string
		params   map[string]interface{}
		expected error
	}

	testTable := []testTableData{
		{
			tcase: "correct params",
			params: map[string]interface{}{
				"endpoint": "http://jenkins.company.com/",
				"job":      "SomeJob",
				"login":    "admin",
				"password": "qwerty123",
			},
			expected: nil,
		},
		{
			tcase: "param missing",
			params: map[string]interface{}{
				"endpoint": "http://jenkins.company.com/",
				"job":      "SomeJob",
				"login":    "admin",
			},
			expected: errors.New("required parameter password is missing"),
		},
		{
			tcase: "param wrong type",
			params: map[string]interface{}{
				"endpoint": "http://jenkins.company.com/",
				"job":      "SomeJob",
				"login":    123,
				"password": "qwerty123",
			},
			expected: errors.New("login parameter value is not a string"),
		},
		{
			tcase: "param wrong type",
			params: map[string]interface{}{
				"endpoint":            "http://jenkins.company.com/",
				"job":                 "SomeJob",
				"login":               "admin",
				"password":            "qwerty123",
				"job parameter wrong": 123,
			},
			expected: errors.New("job parameter wrong parameter value is not a string"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, executorMock.ValidateParameters(testUnit.params), testUnit.tcase)
	}
}

func TestJenkinsTaskExecutor_NewTask(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor()

	type testTableData struct {
		tcase                string
		eventID, rule, alert string
		blockTTL             time.Duration
		preparedParameters   map[string]interface{}
		expected             func() executor.Task
	}

	testTable := []testTableData{
		{
			tcase:    "all params",
			eventID:  "825e",
			rule:     "testrule1",
			alert:    "testalert1",
			blockTTL: 1 * time.Second,
			preparedParameters: map[string]interface{}{
				"endpoint":                              "http://jenkins.company.com/",
				"job":                                   "SomeJob",
				"login":                                 "admin",
				"password":                              "qwerty123",
				"state_refresh_delay":                   "1m",
				"secure_interations_limit":              666,
				"job parameter test":                    "test1",
				"job parameter test job parameter test": "test2",
			},
			expected: func() executor.Task {
				task := &task{
					jenkins: gojenkins.CreateJenkins(
						nil,
						"http://jenkins.company.com/",
						"admin",
						"qwerty123",
					),
				}
				task.job = "SomeJob"
				task.stateRefreshDelay = 1 * time.Minute
				task.secureInterationsLimit = 666
				task.secureBuildDelay = defaultSecureBuildDelay
				task.parameters = map[string]string{
					"test":                    "test1",
					"test job parameter test": "test2",
				}
				task.SetBase("825e", "testrule1", "testalert1", 1*time.Second)
				return task
			},
		},
		{
			tcase:    "default params + no extra params",
			eventID:  "825e",
			rule:     "testrule1",
			alert:    "testalert1",
			blockTTL: 1 * time.Second,
			preparedParameters: map[string]interface{}{
				"endpoint": "http://jenkins.company.com/",
				"job":      "SomeJob",
				"login":    "admin",
				"password": "qwerty123",
			},
			expected: func() executor.Task {
				task := &task{
					jenkins: gojenkins.CreateJenkins(
						nil,
						"http://jenkins.company.com/",
						"admin",
						"qwerty123",
					),
				}
				task.job = "SomeJob"
				task.stateRefreshDelay = defaultStateRefreshDelay
				task.secureInterationsLimit = defaultSecureInterationsLimit
				task.parameters = map[string]string{}
				task.secureBuildDelay = defaultSecureBuildDelay
				task.SetBase("825e", "testrule1", "testalert1", 1*time.Second)
				return task
			},
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected(), executorMock.NewTask(testUnit.eventID, testUnit.rule, testUnit.alert, testUnit.blockTTL, testUnit.preparedParameters), testUnit.tcase)
	}
}

func TestGetBuildID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jenkinsMock := NewMockJenkins(ctrl)

	type testTableData struct {
		tcase           string
		job             string
		queueID         int64
		expectFunc      func(j *MockJenkins, jb string, qID int64)
		expectedBuildID int64
		expectedErr     error
	}

	testTable := []testTableData{
		{
			tcase:   "3 loops",
			job:     "test",
			queueID: 10,
			expectFunc: func(j *MockJenkins, jb string, qID int64) {
				j.EXPECT().GetAllBuildIds(jb).Return([]gojenkins.JobBuild{{Number: 1}, {Number: 2}, {Number: 3}}, nil)
				j.EXPECT().GetBuild(jb, int64(1)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 4}}, nil)
				j.EXPECT().GetBuild(jb, int64(2)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 8}}, nil)
				j.EXPECT().GetBuild(jb, int64(3)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 10}}, nil)
			},
			expectedBuildID: 3,
			expectedErr:     nil,
		},
		{
			tcase:   "3 loops + not found",
			job:     "test",
			queueID: 10,
			expectFunc: func(j *MockJenkins, jb string, qID int64) {
				j.EXPECT().GetAllBuildIds(jb).Return([]gojenkins.JobBuild{{Number: 1}, {Number: 2}, {Number: 3}}, nil)
				j.EXPECT().GetBuild(jb, int64(1)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 4}}, nil)
				j.EXPECT().GetBuild(jb, int64(2)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 6}}, nil)
				j.EXPECT().GetBuild(jb, int64(3)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 8}}, nil)
			},
			expectedBuildID: 0,
			expectedErr:     nil,
		},
		{
			tcase:   "get all builds error",
			job:     "test",
			queueID: 10,
			expectFunc: func(j *MockJenkins, jb string, qID int64) {
				j.EXPECT().GetAllBuildIds(jb).Return(nil, errors.New("get all builds error"))
			},
			expectedBuildID: 0,
			expectedErr:     errors.New("get all builds error"),
		},
		{
			tcase:   "get build error",
			job:     "test",
			queueID: 10,
			expectFunc: func(j *MockJenkins, jb string, qID int64) {
				j.EXPECT().GetAllBuildIds(jb).Return([]gojenkins.JobBuild{{Number: 1}, {Number: 2}, {Number: 3}}, nil)
				j.EXPECT().GetBuild(jb, int64(1)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 4}}, nil)
				j.EXPECT().GetBuild(jb, int64(2)).Return(nil, errors.New("get build error"))
			},
			expectedBuildID: 0,
			expectedErr:     errors.New("get build error"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(jenkinsMock, testUnit.job, testUnit.queueID)
		buildID, err := getBuildID(jenkinsMock, testUnit.job, testUnit.queueID)
		assert.Equal(t, testUnit.expectedBuildID, buildID, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}
}

func TestJenkinsTask_Exec(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jenkinsMock := NewMockJenkins(ctrl)

	logger, hook := test.NewNullLogger()

	task := &task{
		job:                    "SomeJob",
		stateRefreshDelay:      0 * time.Second,
		secureInterationsLimit: 4,
		secureBuildDelay:       0 * time.Second,
		parameters:             map[string]string{"test": "test1"},
		jenkins:                jenkinsMock,
	}
	task.SetBase("id", "rule", "alert", 10*time.Minute)

	type testTableData struct {
		tcase       string
		expectFunc  func(j *MockJenkins)
		expectedErr error
	}

	testTable := []testTableData{
		{
			tcase: "3 loops",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, nil)
				j.EXPECT().BuildJob("SomeJob", map[string]string{"test": "test1"}).Return(int64(10), nil)
				j.EXPECT().GetAllBuildIds("SomeJob").Return([]gojenkins.JobBuild{{Number: 5}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(5)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 1}}, nil)
				j.EXPECT().GetAllBuildIds("SomeJob").Return([]gojenkins.JobBuild{{Number: 5}, {Number: 20}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(5)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 1}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 10}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{Building: true}}, nil).Times(2)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{Building: false, Result: gojenkins.STATUS_SUCCESS}}, nil)
			},
			expectedErr: nil,
		},
		{
			tcase: "build failed",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, nil)
				j.EXPECT().BuildJob("SomeJob", map[string]string{"test": "test1"}).Return(int64(10), nil)
				j.EXPECT().GetAllBuildIds("SomeJob").Return([]gojenkins.JobBuild{{Number: 20}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 10}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{Building: false, Result: gojenkins.STATUS_FAIL}}, nil)
			},
			expectedErr: errors.New("build failed"),
		},
		{
			tcase: "get build error",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, nil)
				j.EXPECT().BuildJob("SomeJob", map[string]string{"test": "test1"}).Return(int64(10), nil)
				j.EXPECT().GetAllBuildIds("SomeJob").Return([]gojenkins.JobBuild{{Number: 20}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(&gojenkins.Build{Raw: &gojenkins.BuildResponse{QueueID: 10}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(nil, errors.New("get build error"))
			},
			expectedErr: errors.New("get build error"),
		},
		{
			tcase: "get build ID error",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, nil)
				j.EXPECT().BuildJob("SomeJob", map[string]string{"test": "test1"}).Return(int64(10), nil)
				j.EXPECT().GetAllBuildIds("SomeJob").Return([]gojenkins.JobBuild{{Number: 20}}, nil)
				j.EXPECT().GetBuild("SomeJob", int64(20)).Return(nil, errors.New("get build ID error"))
			},
			expectedErr: errors.New("get build ID error"),
		},
		{
			tcase: "build job error",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, nil)
				j.EXPECT().BuildJob("SomeJob", map[string]string{"test": "test1"}).Return(int64(0), errors.New("get build ID error"))
			},
			expectedErr: errors.New("get build ID error"),
		},
		{
			tcase: "init error",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, errors.New("init error"))
			},
			expectedErr: errors.New("init error"),
		},
		{
			tcase: "secure iterations limit exceed",
			expectFunc: func(j *MockJenkins) {
				j.EXPECT().Init().Return(nil, nil)
				j.EXPECT().BuildJob("SomeJob", map[string]string{"test": "test1"}).Return(int64(10), nil)
				j.EXPECT().GetAllBuildIds("SomeJob").Return([]gojenkins.JobBuild{}, nil).Times(4)
			},
			expectedErr: errors.New("secure iterations limit exceed"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(jenkinsMock)
		assert.Equal(t, testUnit.expectedErr, task.Exec(logger), testUnit.tcase)
	}

	// logger is not used
	assert.Equal(t, 0, len(hook.Entries))
}
