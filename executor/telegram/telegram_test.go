package telegram

import (
	"errors"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang/mock/gomock"
	"github.com/krpn/prometheus-alert-webhooker/executor"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestTelegramTask_ExecutorInterface(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(&http.Client{})
	task := executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
		map[string]interface{}{
			"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			"chat_id":   12345678,
			"message":   "test",
		},
	)

	type testTableData struct {
		tcase    string
		taskFunc func(t executor.Task) interface{}
		expected interface{}
	}

	testTable := []testTableData{
		{
			tcase: "ExecutorName func",
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorName()
			},
			expected: "telegram",
		},
		{
			tcase: "ExecutorDetails func",
			taskFunc: func(t executor.Task) interface{} {
				return t.ExecutorDetails()
			},
			expected: map[string]interface{}{"chatID": int64(12345678), "message": "test"},
		},
		{
			tcase: "Fingerprint func",
			taskFunc: func(t executor.Task) interface{} {
				return t.Fingerprint()
			},
			expected: "1d5f106baa98b339d1399561f1e38112",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, testUnit.taskFunc(task), testUnit.tcase)
	}
}

func TestTelegramTaskExecutor_NewTask(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(&http.Client{})

	testTask := executorMock.NewTask("825e", "testrule1", "testalert1", 1*time.Second,
		map[string]interface{}{
			"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			"chat_id":   12345678,
			"message":   "test",
		},
	)

	expected := &task{
		chatID:  12345678,
		message: "test",
		telegram: &tgbotapi.BotAPI{
			Token:  "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			Buffer: defaultBuffer,
			Client: &http.Client{},
		},
	}
	expected.SetBase("825e", "testrule1", "testalert1", 1*time.Second)

	assert.Equal(t, expected, testTask)
}

func TestTelegramTaskExecutor_ValidateParameters(t *testing.T) {
	t.Parallel()

	executorMock := NewExecutor(&http.Client{})

	type testTableData struct {
		tcase    string
		params   map[string]interface{}
		expected error
	}

	testTable := []testTableData{
		{
			tcase: "correct params",
			params: map[string]interface{}{
				"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				"chat_id":   12345678,
				"message":   "test",
			},
			expected: nil,
		},
		{
			tcase: "param missing",
			params: map[string]interface{}{
				"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				"message":   "test",
			},
			expected: errors.New("required parameter chat_id is missing"),
		},
		{
			tcase: "param wrong type",
			params: map[string]interface{}{
				"bot_token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				"chat_id":   "12345678",
				"message":   "test",
			},
			expected: errors.New("chat_id parameter value is not int"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, executorMock.ValidateParameters(testUnit.params), testUnit.tcase)
	}
}

func TestTelegramTask_Exec(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	telegramMock := NewMockTelegram(ctrl)

	logger, hook := test.NewNullLogger()

	task := &task{
		chatID:   12345678,
		message:  "test",
		telegram: telegramMock,
	}
	task.SetBase("id", "rule", "alert", 10*time.Minute)

	type testTableData struct {
		tcase       string
		expectFunc  func(t *MockTelegram)
		expectedErr error
	}

	testTable := []testTableData{
		{
			tcase: "success",
			expectFunc: func(t *MockTelegram) {
				t.EXPECT().Send(tgbotapi.NewMessage(12345678, "test")).Return(tgbotapi.Message{}, nil)
			},
			expectedErr: nil,
		},
		{
			tcase: "error",
			expectFunc: func(t *MockTelegram) {
				t.EXPECT().Send(tgbotapi.NewMessage(12345678, "test")).Return(tgbotapi.Message{}, errors.New("error"))
			},
			expectedErr: errors.New("error"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(telegramMock)
		assert.Equal(t, testUnit.expectedErr, task.Exec(logger), testUnit.tcase)
	}

	// logger is not used
	assert.Equal(t, 0, len(hook.Entries))
}
