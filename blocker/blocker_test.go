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
		fingerprint                 string
		expectFunc                  func(m *Mockcacher, fingerprint string)
		expectedBlockedSuccessfully bool
		expectedErr                 error
	}

	testTable := []testTableData{
		{
			tcase: "blocked successfully",
			expectFunc: func(m *Mockcacher, fingerprint string) {
				m.EXPECT().Get([]byte(fingerprint)).Return(nil, freecache.ErrNotFound)
				m.EXPECT().Set([]byte(fingerprint), defValue, foreverTTL).Return(nil)
			},
			expectedBlockedSuccessfully: true,
			expectedErr:                 nil,
		},
		{
			tcase: "block error",
			expectFunc: func(m *Mockcacher, fingerprint string) {
				m.EXPECT().Get([]byte(fingerprint)).Return(nil, freecache.ErrNotFound)
				m.EXPECT().Set([]byte(fingerprint), defValue, foreverTTL).Return(errors.New("set error"))
			},
			expectedBlockedSuccessfully: false,
			expectedErr:                 errors.New("set error"),
		},
		{
			tcase: "already blocked",
			expectFunc: func(m *Mockcacher, fingerprint string) {
				m.EXPECT().Get([]byte(fingerprint)).Return([]byte(defValue), nil)
			},
			expectedBlockedSuccessfully: false,
			expectedErr:                 nil,
		},
		{
			tcase: "check block error",
			expectFunc: func(m *Mockcacher, fingerprint string) {
				m.EXPECT().Get([]byte(fingerprint)).Return(nil, errors.New("get error"))
			},
			expectedBlockedSuccessfully: false,
			expectedErr:                 errors.New("get error"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(cache, testUnit.fingerprint)
		blockedSuccessfully, err := blocker.BlockInProgress(testUnit.fingerprint)
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
		fingerprint string
		ttl         time.Duration
		expectFunc  func(m *Mockcacher, fingerprint string, ttl time.Duration)
		expectedErr error
	}

	testTable := []testTableData{
		{
			tcase:       "success",
			fingerprint: "test",
			ttl:         10 * time.Second,
			expectFunc: func(m *Mockcacher, fingerprint string, ttl time.Duration) {
				m.EXPECT().Set([]byte(fingerprint), []byte(defValue), int(ttl.Seconds())).Return(nil)
			},
			expectedErr: nil,
		},
		{
			tcase:       "error",
			fingerprint: "test",
			ttl:         10 * time.Second,
			expectFunc: func(m *Mockcacher, fingerprint string, ttl time.Duration) {
				m.EXPECT().Set([]byte(fingerprint), []byte(defValue), int(ttl.Seconds())).Return(errors.New("some error"))
			},
			expectedErr: errors.New("some error"),
		},
	}

	for _, testUnit := range testTable {
		testUnit.expectFunc(cache, testUnit.fingerprint, testUnit.ttl)
		err := blocker.BlockForTTL(testUnit.fingerprint, testUnit.ttl)
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
		b, e := blocker.BlockInProgress("test")
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

	cache.EXPECT().Del([]byte("test")).Return(true)

	blocker.Unblock("test")
}
