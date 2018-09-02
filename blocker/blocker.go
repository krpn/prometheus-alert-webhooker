package blocker

import (
	"strings"
	"sync"
	"time"
)

const foreverTTL = 0

// Blocker blocks tasks by fingerprint.
type Blocker struct {
	cache    cacher
	mt       *sync.Mutex
	defValue []byte
}

// BlockInProgress blocks task by fingeprint while executing.
func (b *Blocker) BlockInProgress(fingerprint string) (blockedSuccessfully bool, err error) {
	b.mt.Lock()
	defer b.mt.Unlock()

	res, err := b.cache.Get([]byte(fingerprint))
	if len(res) != 0 {
		return false, nil
	}

	if err != nil && !strings.Contains(err.Error(), "not found") {
		return false, err
	}

	err = b.cache.Set([]byte(fingerprint), b.defValue, foreverTTL)
	if err != nil {
		return false, err
	}

	return true, nil
}

// BlockForTTL blocks task by fingeprint for needed TTL.
func (b *Blocker) BlockForTTL(fingerprint string, ttl time.Duration) error {
	b.mt.Lock()
	defer b.mt.Unlock()

	return b.cache.Set([]byte(fingerprint), b.defValue, int(ttl.Seconds()))
}

// Unblock unblocks task by fingeprint.
func (b *Blocker) Unblock(fingerprint string) {
	b.mt.Lock()
	defer b.mt.Unlock()

	_ = b.cache.Del([]byte(fingerprint))
}

var defValue = []byte("l")

// New creates Blocker instance.
func New(cache cacher) *Blocker {
	return &Blocker{
		cache:    cache,
		mt:       &sync.Mutex{},
		defValue: defValue,
	}
}

//go:generate mockgen -source=blocker.go -destination=blocker_mocks.go -package=blocker doc github.com/golang/mock/gomock

type cacher interface {
	Get(key []byte) (value []byte, err error)
	Set(key, value []byte, expireSeconds int) (err error)
	Del(key []byte) (affected bool)
}
