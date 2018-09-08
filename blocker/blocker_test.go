package blocker

import (
	"errors"
	"github.com/coocood/freecache"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBlocker_BlockInProgress(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := NewMockcacher(ctrl)
	blocker := New(cache)

	type testTableData struct {
		tcase                       string
		executor                    string
		fingerprint                 string
		expectFunc                  func(m *Mockcacher, key []byte)
		expectedBlockedSuccessfully bool
		expectedErr                 error
	}

	testTable := []testTableData{
		{
			tcase:       "blocked successfully",
			executor:    "jenkins",
			fingerprint: "test",
			expectFunc: func(m *Mockcacher, key []byte) {
				m.EXPECT().Get(key).Return(nil, freecache.ErrNotFound)
				m.EXPECT().Set(key, []byte(defValue), foreverTTL).Return(nil)
			},
			expectedBlockedSuccessfully: true,
			expectedErr:                 nil,
		},
		{
			tcase:       "block error",
			executor:    "jenkins",
			fingerprint: "test",
			expectFunc: func(m *Mockcacher, key []byte) {
				m.EXPECT().Get(key).Return(nil, freecache.ErrNotFound)
				m.EXPECT().Set(key, []byte(defValue), foreverTTL).Return(errors.New("set error"))
			},
			expectedBlockedSuccessfully: false,
			expectedErr:                 errors.New("set error"),
		},
		{
			tcase:       "already blocked",
			executor:    "jenkins",
			fingerprint: "test",
			expectFunc: func(m *Mockcacher, key []byte) {
				m.EXPECT().Get(key).Return([]byte(defValue), nil)
			},
			expectedBlockedSuccessfully: false,
			expectedErr:                 nil,
		},
		{
			tcase:       "check block error",
			executor:    "jenkins",
			fingerprint: "test",
			expectFunc: func(m *Mockcacher, key []byte) {
				m.EXPECT().Get(key).Return(nil, errors.New("get error"))
			},
			expectedBlockedSuccessfully: false,
			expectedErr:                 errors.New("get error"),
		},
	}

	for _, testUnit := range testTable {
		key := getBlockKey(testUnit.executor, testUnit.fingerprint)
		testUnit.expectFunc(cache, key)
		blockedSuccessfully, err := blocker.BlockInProgress(testUnit.executor, testUnit.fingerprint)
		assert.Equal(t, testUnit.expectedBlockedSuccessfully, blockedSuccessfully, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}
}

func TestBlocker_BlockForTTL(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := NewMockcacher(ctrl)
	blocker := New(cache)

	type testTableData struct {
		tcase       string
		executor    string
		fingerprint string
		ttl         time.Duration
		expectFunc  func(m *Mockcacher, key []byte, ttl time.Duration)
		expectedErr error
	}

	testTable := []testTableData{
		{
			tcase:       "success",
			executor:    "jenkins",
			fingerprint: "test",
			ttl:         10 * time.Second,
			expectFunc: func(m *Mockcacher, key []byte, ttl time.Duration) {
				m.EXPECT().Set(key, []byte(defValue), int(ttl.Seconds())).Return(nil)
			},
			expectedErr: nil,
		},
		{
			tcase:       "error",
			executor:    "jenkins",
			fingerprint: "test",
			ttl:         10 * time.Second,
			expectFunc: func(m *Mockcacher, key []byte, ttl time.Duration) {
				m.EXPECT().Set(key, []byte(defValue), int(ttl.Seconds())).Return(errors.New("some error"))
			},
			expectedErr: errors.New("some error"),
		},
	}

	for _, testUnit := range testTable {
		key := getBlockKey(testUnit.executor, testUnit.fingerprint)
		testUnit.expectFunc(cache, key, testUnit.ttl)
		err := blocker.BlockForTTL(testUnit.executor, testUnit.fingerprint, testUnit.ttl)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}
}

func TestBlocker_BlockAsync(t *testing.T) {
	t.Parallel()

	blocker := New(freecache.NewCache(1 * 1024 * 1024))

	interations := 1000

	blockedCh := make(chan bool, interations)
	errCh := make(chan error, interations)

	ablock := func(bCh chan bool, eCh chan error) {
		b, e := blocker.BlockInProgress("jenkins", "test")
		bCh <- b
		eCh <- e
	}

	for i := 0; i < interations; i++ {
		go ablock(blockedCh, errCh)
	}

	var blockedCounter, errCounter int
	for i := 0; i < interations; i++ {
		if <-blockedCh {
			blockedCounter++
		}

		if <-errCh != nil {
			errCounter++
		}
	}

	if blockedCounter != 1 {
		t.Error("block must be one")
	}

	if errCounter != 0 {
		t.Error("errors occurred")
	}
}

func TestBlocker_Unblock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := NewMockcacher(ctrl)
	blocker := New(cache)

	key := getBlockKey("jenkins", "test")

	cache.EXPECT().Del(key).Return(true)

	blocker.Unblock("jenkins", "test")
}

func Test_GetKey(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		executor    string
		fingerprint string
		expected    []byte
	}

	testTable := []testTableData{
		{
			executor:    "jenkins",
			fingerprint: "test",
			expected:    []byte("jenkins_test"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, getBlockKey(testUnit.executor, testUnit.fingerprint), testUnit.expected)
	}
}
