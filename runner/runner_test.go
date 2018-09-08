package runner

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/krpn/prometheus-alert-webhooker/model"
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
		tasks      []*executor.MockTask
		expectFunc func(ts []*executor.MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger)
	}

	type testTableData struct {
		tcase        string
		tasks        []expectTask
		expectedLogs []string
	}

	testTable := []testTableData{
		{
			tcase: "a lot of tasks",
			tasks: []expectTask{
				{
					tasks: []*executor.MockTask{executor.NewMockTask(ctrl)},
					expectFunc: func(ts []*executor.MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						for _, t := range ts {
							t.EXPECT().BlockTTL().Return(0 * time.Second)
							t.EXPECT().Exec(l).Return(nil)
							t.EXPECT().EventID().Return("testid1").Times(2)
							t.EXPECT().Rule().Return("testrule1").Times(3)
							t.EXPECT().Alert().Return("testalert1").Times(3)
							t.EXPECT().ExecutorName().Return("shell").Times(3)
							t.EXPECT().ExecutorDetails().Return(map[string]string{"testtask1": "opts"}).Times(2)
							m.EXPECT().ExecutedTaskObserve("testrule1", "testalert1", "shell", execResultSuccessWithoutBlock.String(), nil, 0*time.Second)
						}
					},
				},
				{
					tasks: []*executor.MockTask{executor.NewMockTask(ctrl)},
					expectFunc: func(ts []*executor.MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						for _, t := range ts {
							t.EXPECT().BlockTTL().Return(10 * time.Minute)
							t.EXPECT().Fingerprint().Return("testfp2")
							t.EXPECT().ExecutorName().Return("shell").Times(4)
							b.EXPECT().BlockInProgress("shell", "testfp2").Return(false, nil)
							t.EXPECT().EventID().Return("testid2").Times(2)
							t.EXPECT().Rule().Return("testrule2").Times(3)
							t.EXPECT().Alert().Return("testalert2").Times(3)
							t.EXPECT().ExecutorDetails().Return("testtask2").Times(2)
							m.EXPECT().ExecutedTaskObserve("testrule2", "testalert2", "shell", execResultInBlock.String(), nil, 0*time.Second)
						}
					},
				},
				{
					tasks: []*executor.MockTask{executor.NewMockTask(ctrl)},
					expectFunc: func(ts []*executor.MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						for _, t := range ts {
							t.EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
							t.EXPECT().Fingerprint().Return("testfp3").Times(2)
							t.EXPECT().ExecutorName().Return("shell").Times(5)
							b.EXPECT().BlockInProgress("shell", "testfp3").Return(true, nil)
							t.EXPECT().Exec(l).Return(nil)
							b.EXPECT().BlockForTTL("shell", "testfp3", 10*time.Minute).Return(nil)
							t.EXPECT().EventID().Return("testid3").Times(2)
							t.EXPECT().Rule().Return("testrule3").Times(3)
							t.EXPECT().Alert().Return("testalert3").Times(3)
							t.EXPECT().ExecutorDetails().Return("testtask3").Times(2)
							m.EXPECT().ExecutedTaskObserve("testrule3", "testalert3", "shell", execResultSuccess.String(), nil, 0*time.Second)
						}
					},
				},
				{
					tasks: []*executor.MockTask{executor.NewMockTask(ctrl), executor.NewMockTask(ctrl)},
					expectFunc: func(ts []*executor.MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						for i, t := range ts {
							if i == 1 {
								// second will not exec because of error
								t.EXPECT().EventID().Return("testid5").Times(1)
								t.EXPECT().Rule().Return("testrule5").Times(1)
								t.EXPECT().Alert().Return("testalert5").Times(1)
								t.EXPECT().ExecutorName().Return("shell").Times(1)
								t.EXPECT().ExecutorDetails().Return("testtask5").Times(1)
								continue
							}

							t.EXPECT().BlockTTL().Return(10 * time.Minute)
							t.EXPECT().Fingerprint().Return("testfp4").Times(2)
							t.EXPECT().ExecutorName().Return("shell").Times(5)
							b.EXPECT().BlockInProgress("shell", "testfp4").Return(true, nil)
							t.EXPECT().Exec(l).Return(errors.New("exec error"))
							b.EXPECT().Unblock("shell", "testfp4")
							t.EXPECT().EventID().Return("testid4").Times(2)
							t.EXPECT().Rule().Return("testrule4").Times(3)
							t.EXPECT().Alert().Return("testalert4").Times(3)
							t.EXPECT().ExecutorDetails().Return("testtask4").Times(2)
							m.EXPECT().ExecutedTaskObserve("testrule4", "testalert4", "shell", execResultExecError.String(), errors.New("exec error"), 0*time.Second)
						}
					},
				},
				{
					tasks: []*executor.MockTask{executor.NewMockTask(ctrl), executor.NewMockTask(ctrl), executor.NewMockTask(ctrl)},
					expectFunc: func(ts []*executor.MockTask, b *Mockblocker, m *Mockmetricser, l *logrus.Logger) {
						shift := 6

						i := 0
						ts[i].EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
						ts[i].EXPECT().Fingerprint().Return(fmt.Sprintf("testfp%v", i+shift)).Times(2)
						ts[i].EXPECT().ExecutorName().Return("shell").Times(2)
						b.EXPECT().BlockInProgress("shell", fmt.Sprintf("testfp%v", i+shift)).Return(true, nil)
						ts[i].EXPECT().Exec(l).Return(nil)
						b.EXPECT().BlockForTTL("shell", fmt.Sprintf("testfp%v", i+shift), 10*time.Minute).Return(nil)
						ts[i].EXPECT().EventID().Return(fmt.Sprintf("testid%v", i+shift)).Times(2)
						ts[i].EXPECT().Rule().Return(fmt.Sprintf("testrule%v", i+shift)).Times(3)
						ts[i].EXPECT().Alert().Return(fmt.Sprintf("testalert%v", i+shift)).Times(3)
						ts[i].EXPECT().ExecutorName().Return("shell").Times(3)
						ts[i].EXPECT().ExecutorDetails().Return(fmt.Sprintf("testtask%v", i+shift)).Times(2)
						m.EXPECT().ExecutedTaskObserve(fmt.Sprintf("testrule%v", i+shift), fmt.Sprintf("testalert%v", i+shift), "shell", execResultSuccess.String(), nil, 0*time.Second)

						i = 1
						ts[i].EXPECT().BlockTTL().Return(10 * time.Minute).Times(1)
						ts[i].EXPECT().Fingerprint().Return(fmt.Sprintf("testfp%v", i+shift))
						ts[i].EXPECT().ExecutorName().Return("shell")
						b.EXPECT().BlockInProgress("shell", fmt.Sprintf("testfp%v", i+shift)).Return(false, nil)
						ts[i].EXPECT().EventID().Return(fmt.Sprintf("testid%v", i+shift)).Times(2)
						ts[i].EXPECT().Rule().Return(fmt.Sprintf("testrule%v", i+shift)).Times(3)
						ts[i].EXPECT().Alert().Return(fmt.Sprintf("testalert%v", i+shift)).Times(3)
						ts[i].EXPECT().ExecutorName().Return("shell").Times(3)
						ts[i].EXPECT().ExecutorDetails().Return(fmt.Sprintf("testtask%v", i+shift)).Times(2)
						m.EXPECT().ExecutedTaskObserve(fmt.Sprintf("testrule%v", i+shift), fmt.Sprintf("testalert%v", i+shift), "shell", execResultInBlock.String(), nil, 0*time.Second)

						i = 2
						ts[i].EXPECT().EventID().Return(fmt.Sprintf("testid%v", i+shift)).Times(1)
						ts[i].EXPECT().Rule().Return(fmt.Sprintf("testrule%v", i+shift)).Times(1)
						ts[i].EXPECT().Alert().Return(fmt.Sprintf("testalert%v", i+shift)).Times(1)
						ts[i].EXPECT().ExecutorName().Return("shell").Times(1)
						ts[i].EXPECT().ExecutorDetails().Return(fmt.Sprintf("testtask%v", i+shift)).Times(1)
					},
				},
			},
			expectedLogs: []string{
				`{"context":"runner","level":"debug","msg":"runner starts executing group","tasks":[{"alert":"testalert1","details":{"testtask1":"opts"},"event_id":"testid1","executor":"shell","rule":"testrule1"}]}`,
				`{"context":"runner","level":"debug","msg":"runner starts executing group","tasks":[{"alert":"testalert2","details":"testtask2","event_id":"testid2","executor":"shell","rule":"testrule2"}]}`,
				`{"context":"runner","level":"debug","msg":"runner starts executing group","tasks":[{"alert":"testalert3","details":"testtask3","event_id":"testid3","executor":"shell","rule":"testrule3"}]}`,
				`{"context":"runner","level":"debug","msg":"runner starts executing group","tasks":[{"alert":"testalert4","details":"testtask4","event_id":"testid4","executor":"shell","rule":"testrule4"},{"alert":"testalert5","details":"testtask5","event_id":"testid5","executor":"shell","rule":"testrule5"}]}`,
				`{"context":"runner","level":"debug","msg":"runner starts executing group","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
				`{"alert":"testalert1","context":"runner","details":{"testtask1":"opts"},"event_id":"testid1","executor":"shell","level":"debug","msg":"runner starts executing task #1/1","rule":"testrule1","tasks":[{"alert":"testalert1","details":{"testtask1":"opts"},"event_id":"testid1","executor":"shell","rule":"testrule1"}]}`,
				`{"alert":"testalert1","context":"runner","details":{"testtask1":"opts"},"duration":"0s","event_id":"testid1","executor":"shell","level":"debug","msg":"runner finished executing task #1/1","result":"success_without_block","rule":"testrule1","tasks":[{"alert":"testalert1","details":{"testtask1":"opts"},"event_id":"testid1","executor":"shell","rule":"testrule1"}]}`,
				`{"alert":"testalert2","context":"runner","details":"testtask2","event_id":"testid2","executor":"shell","level":"debug","msg":"runner starts executing task #1/1","rule":"testrule2","tasks":[{"alert":"testalert2","details":"testtask2","event_id":"testid2","executor":"shell","rule":"testrule2"}]}`,
				`{"alert":"testalert2","context":"runner","details":"testtask2","duration":"0s","event_id":"testid2","executor":"shell","level":"debug","msg":"runner finished executing task #1/1","result":"in_block","rule":"testrule2","tasks":[{"alert":"testalert2","details":"testtask2","event_id":"testid2","executor":"shell","rule":"testrule2"}]}`,
				`{"alert":"testalert2","context":"runner","details":"testtask2","duration":"0s","event_id":"testid2","executor":"shell","level":"debug","msg":"runner got executing task #1/1 unsuccessful result, stopping group: in_block","result":"in_block","rule":"testrule2","tasks":[{"alert":"testalert2","details":"testtask2","event_id":"testid2","executor":"shell","rule":"testrule2"}]}`,
				`{"alert":"testalert3","context":"runner","details":"testtask3","event_id":"testid3","executor":"shell","level":"debug","msg":"runner starts executing task #1/1","rule":"testrule3","tasks":[{"alert":"testalert3","details":"testtask3","event_id":"testid3","executor":"shell","rule":"testrule3"}]}`,
				`{"alert":"testalert3","context":"runner","details":"testtask3","duration":"0s","event_id":"testid3","executor":"shell","level":"debug","msg":"runner finished executing task #1/1","result":"success","rule":"testrule3","tasks":[{"alert":"testalert3","details":"testtask3","event_id":"testid3","executor":"shell","rule":"testrule3"}]}`,
				`{"alert":"testalert4","context":"runner","details":"testtask4","event_id":"testid4","executor":"shell","level":"debug","msg":"runner starts executing task #1/2","rule":"testrule4","tasks":[{"alert":"testalert4","details":"testtask4","event_id":"testid4","executor":"shell","rule":"testrule4"},{"alert":"testalert5","details":"testtask5","event_id":"testid5","executor":"shell","rule":"testrule5"}]}`,
				`{"alert":"testalert4","context":"runner","details":"testtask4","duration":"0s","event_id":"testid4","executor":"shell","level":"error","msg":"runner got executing task #1/2 error, stopping group: exec error","result":"exec_error","rule":"testrule4","tasks":[{"alert":"testalert4","details":"testtask4","event_id":"testid4","executor":"shell","rule":"testrule4"},{"alert":"testalert5","details":"testtask5","event_id":"testid5","executor":"shell","rule":"testrule5"}]}`,
				`{"alert":"testalert6","context":"runner","details":"testtask6","event_id":"testid6","executor":"shell","level":"debug","msg":"runner starts executing task #1/3","rule":"testrule6","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
				`{"alert":"testalert6","context":"runner","details":"testtask6","duration":"0s","event_id":"testid6","executor":"shell","level":"debug","msg":"runner finished executing task #1/3","result":"success","rule":"testrule6","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
				`{"alert":"testalert7","context":"runner","details":"testtask7","event_id":"testid7","executor":"shell","level":"debug","msg":"runner starts executing task #2/3","rule":"testrule7","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
				`{"alert":"testalert7","context":"runner","details":"testtask7","duration":"0s","event_id":"testid7","executor":"shell","level":"debug","msg":"runner finished executing task #2/3","result":"in_block","rule":"testrule7","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
				`{"alert":"testalert7","context":"runner","details":"testtask7","duration":"0s","event_id":"testid7","executor":"shell","level":"debug","msg":"runner got executing task #2/3 unsuccessful result, stopping group: in_block","result":"in_block","rule":"testrule7","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
				`{"context":"runner","level":"debug","msg":"runner finished executing group","tasks":[{"alert":"testalert1","details":{"testtask1":"opts"},"event_id":"testid1","executor":"shell","rule":"testrule1"}]}`,
				`{"context":"runner","level":"debug","msg":"runner finished executing group","tasks":[{"alert":"testalert2","details":"testtask2","event_id":"testid2","executor":"shell","rule":"testrule2"}]}`,
				`{"context":"runner","level":"debug","msg":"runner finished executing group","tasks":[{"alert":"testalert3","details":"testtask3","event_id":"testid3","executor":"shell","rule":"testrule3"}]}`,
				`{"context":"runner","level":"debug","msg":"runner finished executing group","tasks":[{"alert":"testalert4","details":"testtask4","event_id":"testid4","executor":"shell","rule":"testrule4"},{"alert":"testalert5","details":"testtask5","event_id":"testid5","executor":"shell","rule":"testrule5"}]}`,
				`{"context":"runner","level":"debug","msg":"runner finished executing group","tasks":[{"alert":"testalert6","details":"testtask6","event_id":"testid6","executor":"shell","rule":"testrule6"},{"alert":"testalert7","details":"testtask7","event_id":"testid7","executor":"shell","rule":"testrule7"},{"alert":"testalert8","details":"testtask8","event_id":"testid8","executor":"shell","rule":"testrule8"}]}`,
			},
		},
	}

	for _, testUnit := range testTable {
		logger, hook := test.NewNullLogger()
		logger.SetLevel(logrus.DebugLevel)
		logger.Formatter = &logrus.JSONFormatter{DisableTimestamp: true}

		tasksCh := make(chan model.Tasks, len(testUnit.tasks))
		for _, task := range testUnit.tasks {
			task.expectFunc(task.tasks, blocker, metric, logger)

			taskGroups := make(model.Tasks, len(task.tasks))
			for j, taskGroup := range task.tasks {
				taskGroups[j] = taskGroup
			}
			tasksCh <- taskGroups
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
