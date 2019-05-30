package runner

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_exec(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	blocker := NewMockblocker(ctrl)
	logger, hook := test.NewNullLogger()

	type testTableData struct {
		tcase          execResult
		task           *executor.MockTask
		expectFunc     func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger)
		expectedResult execResult
		expectedErr    error
	}

	testTable := []testTableData{
		{
			tcase: execResultSuccess,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
				t.EXPECT().Fingerprint().Return("testfp1").Times(2)
				t.EXPECT().ExecutorName().Return("shell").Times(2)
				b.EXPECT().BlockInProgress("shell", "testfp1").Return(true, nil)
				t.EXPECT().Exec(l).Return(nil)
				b.EXPECT().BlockForTTL("shell", "testfp1", 10*time.Minute).Return(nil)
			},
			expectedResult: execResultSuccess,
			expectedErr:    nil,
		},
		{
			tcase: execResultInBlock,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(10 * time.Minute)
				t.EXPECT().Fingerprint().Return("testfp1")
				t.EXPECT().ExecutorName().Return("shell")
				b.EXPECT().BlockInProgress("shell", "testfp1").Return(false, nil)
			},
			expectedResult: execResultInBlock,
			expectedErr:    nil,
		},
		{
			tcase: execResultBlockError,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(10 * time.Minute)
				t.EXPECT().Fingerprint().Return("testfp1")
				t.EXPECT().ExecutorName().Return("shell")
				b.EXPECT().BlockInProgress("shell", "testfp1").Return(false, errors.New("block error"))
			},
			expectedResult: execResultBlockError,
			expectedErr:    errors.New("block error"),
		},
		{
			tcase: execResultExecError,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(10 * time.Minute)
				t.EXPECT().Fingerprint().Return("testfp1").Times(2)
				t.EXPECT().ExecutorName().Return("shell").Times(2)
				b.EXPECT().BlockInProgress("shell", "testfp1").Return(true, nil)
				t.EXPECT().Exec(l).Return(errors.New("exec error"))
				b.EXPECT().Unblock("shell", "testfp1")
			},
			expectedResult: execResultExecError,
			expectedErr:    errors.New("exec error"),
		},
		{
			tcase: execResultSuccessWithoutBlock,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(0 * time.Minute)
				t.EXPECT().Exec(l).Return(nil)
			},
			expectedResult: execResultSuccessWithoutBlock,
			expectedErr:    nil,
		},
		{
			tcase: execResultExecErrorWithoutBlock,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(0 * time.Minute)
				t.EXPECT().Exec(l).Return(errors.New("exec error"))
			},
			expectedResult: execResultExecErrorWithoutBlock,
			expectedErr:    errors.New("exec error"),
		},
		{
			tcase: execResultCanNotBlock,
			task:  executor.NewMockTask(ctrl),
			expectFunc: func(t *executor.MockTask, b *Mockblocker, l *logrus.Logger) {
				t.EXPECT().BlockTTL().Return(10 * time.Minute).Times(2)
				t.EXPECT().Fingerprint().Return("testfp1").Times(2)
				t.EXPECT().ExecutorName().Return("shell").Times(2)
				b.EXPECT().BlockInProgress("shell", "testfp1").Return(true, nil)
				t.EXPECT().Exec(l).Return(nil)
				b.EXPECT().BlockForTTL("shell", "testfp1", 10*time.Minute).Return(errors.New("some block error"))
			},
			expectedResult: execResultCanNotBlock,
			expectedErr:    errors.New("some block error"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(testUnit.task, blocker, logger)
		result, err := exec(testUnit.task, blocker, logger)
		assert.Equal(t, testUnit.expectedResult, result, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}

	assert.Equal(t, 0, len(hook.Entries))
}

func TestExecResult_String(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    execResult
		expected string
	}

	testTable := []testTableData{
		{
			tcase:    execResultSuccess,
			expected: "success",
		},
		{
			tcase:    execResultInBlock,
			expected: "in_block",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.tcase.String(), testUnit.tcase.String())
	}
}
