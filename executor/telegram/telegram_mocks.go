// Code generated by MockGen. DO NOT EDIT.
// Source: telegram.go

// Package telegram is a generated GoMock package.
package telegram

import (
	telegram_bot_api "github.com/go-telegram-bot-api/telegram-bot-api"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockTelegram is a mock of Telegram interface
type MockTelegram struct {
	ctrl     *gomock.Controller
	recorder *MockTelegramMockRecorder
}

// MockTelegramMockRecorder is the mock recorder for MockTelegram
type MockTelegramMockRecorder struct {
	mock *MockTelegram
}

// NewMockTelegram creates a new mock instance
func NewMockTelegram(ctrl *gomock.Controller) *MockTelegram {
	mock := &MockTelegram{ctrl: ctrl}
	mock.recorder = &MockTelegramMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTelegram) EXPECT() *MockTelegramMockRecorder {
	return m.recorder
}

// Send mocks base method
func (m *MockTelegram) Send(c telegram_bot_api.Chattable) (telegram_bot_api.Message, error) {
	ret := m.ctrl.Call(m, "Send", c)
	ret0, _ := ret[0].(telegram_bot_api.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Send indicates an expected call of Send
func (mr *MockTelegramMockRecorder) Send(c interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockTelegram)(nil).Send), c)
}
