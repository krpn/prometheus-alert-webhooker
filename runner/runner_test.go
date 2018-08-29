package runner

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	blocker := NewMockblocker(ctrl)
	metric := NewMockmetricser(ctrl)

	nowFunc := func() time.Time {
		return time.Unix(1535086351, 0)
	}

	type expectTask struct {
		task       *MockTask
		expectFunc func(task *MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger)
	}

	type testTableData struct {
		tcase        string
		tasks        []expectTask
		expectedLogs []string
	}

	testTable := []testTableData{
		{
			tcase: "four tasks",
			tasks: []expectTask{
				{
					task: NewMockTask(ctrl),
					expectFunc: func(t *MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						t.EXPECT().BlockTTL().Return(0 * time.Second)
						t.EXPECT().Exec(l).Return(nil)
						t.EXPECT().EventID().Return("testid1")
						t.EXPECT().Rule().Return("testrule1").Times(2)
						t.EXPECT().Alert().Return("testalert1").Times(2)
						t.EXPECT().ExecutorName().Return("shell").Times(2)
						t.EXPECT().ExecutorDetails().Return(map[string]string{"testtask1": "opts"})
						m.EXPECT().ExecutedTaskObserve("testrule1", "testalert1", "shell", execResultSuccessWithoutBlock.String(), nil, 0*time.Second)
					},
				},
				{
					task: NewMockTask(ctrl),
					expectFunc: func(t *MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						t.EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
						t.EXPECT().Fingerprint().Return("testfp2")
						b.EXPECT().Block("testfp2", 10*time.Minute).Return(false, nil)
						t.EXPECT().EventID().Return("testid2")
						t.EXPECT().Rule().Return("testrule2").Times(2)
						t.EXPECT().Alert().Return("testalert2").Times(2)
						t.EXPECT().ExecutorName().Return("shell").Times(2)
						t.EXPECT().ExecutorDetails().Return("testtask2")
						m.EXPECT().ExecutedTaskObserve("testrule2", "testalert2", "shell", execResultInBlock.String(), nil, 0*time.Second)
					},
				},
				{
					task: NewMockTask(ctrl),
					expectFunc: func(t *MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						t.EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
						t.EXPECT().Fingerprint().Return("testfp3")
						b.EXPECT().Block("testfp3", 10*time.Minute).Return(true, nil)
						t.EXPECT().Exec(l).Return(nil)
						t.EXPECT().EventID().Return("testid3")
						t.EXPECT().Rule().Return("testrule3").Times(2)
						t.EXPECT().Alert().Return("testalert3").Times(2)
						t.EXPECT().ExecutorName().Return("shell").Times(2)
						t.EXPECT().ExecutorDetails().Return("testtask3")
						m.EXPECT().ExecutedTaskObserve("testrule3", "testalert3", "shell", execResultSuccess.String(), nil, 0*time.Second)
					},
				},
				{
					task: NewMockTask(ctrl),
					expectFunc: func(t *MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						t.EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
						t.EXPECT().Fingerprint().Return("testfp4").Times(2)
						b.EXPECT().Block("testfp4", 10*time.Minute).Return(true, nil)
						t.EXPECT().Exec(l).Return(errors.New("exec error"))
						b.EXPECT().Unblock("testfp4")
						t.EXPECT().EventID().Return("testid4")
						t.EXPECT().Rule().Return("testrule4").Times(2)
						t.EXPECT().Alert().Return("testalert4").Times(2)
						t.EXPECT().ExecutorName().Return("shell").Times(2)
						t.EXPECT().ExecutorDetails().Return("testtask4")
						m.EXPECT().ExecutedTaskObserve("testrule4", "testalert4", "shell", execResultExecError.String(), errors.New("exec error"), 0*time.Second)
					},
				},
			},
			expectedLogs: []string{
				`{"alert":"testalert1","context":"runner","details":{"testtask1":"opts"},"event_id":"testid1","executor":"shell","level":"info","msg":"runner starts executing","rule":"testrule1"}`,
				`{"alert":"testalert1","context":"runner","details":{"testtask1":"opts"},"duration":"0s","event_id":"testid1","executor":"shell","level":"info","msg":"runner finished executing","result":"success_without_block","rule":"testrule1"}`,
				`{"alert":"testalert2","context":"runner","details":"testtask2","event_id":"testid2","executor":"shell","level":"info","msg":"runner starts executing","rule":"testrule2"}`,
				`{"alert":"testalert2","context":"runner","details":"testtask2","duration":"0s","event_id":"testid2","executor":"shell","level":"info","msg":"runner finished executing","result":"in_block","rule":"testrule2"}`,
				`{"alert":"testalert3","context":"runner","details":"testtask3","event_id":"testid3","executor":"shell","level":"info","msg":"runner starts executing","rule":"testrule3"}`,
				`{"alert":"testalert3","context":"runner","details":"testtask3","duration":"0s","event_id":"testid3","executor":"shell","level":"info","msg":"runner finished executing","result":"success","rule":"testrule3"}`,
				`{"alert":"testalert4","context":"runner","details":"testtask4","event_id":"testid4","executor":"shell","level":"info","msg":"runner starts executing","rule":"testrule4"}`,
				`{"alert":"testalert4","context":"runner","details":"testtask4","duration":"0s","event_id":"testid4","executor":"shell","level":"error","msg":"runner got executing error: exec error","result":"exec_error","rule":"testrule4"}`,
			},
		},
	}

	for _, testUnit := range testTable {
		logger, hook := test.NewNullLogger()
		logger.Formatter = &logrus.JSONFormatter{DisableTimestamp: true}

		tasksCh := make(chan executor.Task, len(testUnit.tasks))
		for _, task := range testUnit.tasks {
			task.expectFunc(task.task, blocker, metric, logger)
			tasksCh <- task.task
		}
		close(tasksCh)
		Start(len(testUnit.tasks), tasksCh, blocker, metric, logger, nowFunc)

		logs := logsFromHook(t, hook)
		expectedLogs := expectedLogsFix(testUnit.expectedLogs)

		// order can change due to async executing
		sort.Strings(logs)
		sort.Strings(expectedLogs)

		assert.Equal(t, expectedLogs, logs, testUnit.tcase)
	}
}

func logsFromHook(t *testing.T, hook *test.Hook) (logs []string) {
	if hook == nil {
		return []string{}
	}

	if hook.Entries == nil {
		return []string{}
	}

	logs = make([]string, len(hook.Entries))
	for i, entry := range hook.Entries {
		log, err := entry.String()
		assert.Equal(t, nil, err)
		logs[i] = log
	}
	return
}

func expectedLogsFix(logs []string) (expectedLogs []string) {
	expectedLogs = make([]string, len(logs))
	for i, log := range logs {
		expectedLogs[i] = log + "\n"
	}
	return
}
