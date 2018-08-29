package blocker

import (
	"strings"
	"sync"
	"time"
)

// Blocker blocks tasks by fingerprint.
type Blocker struct {
	cache    cacher
	mt       *sync.Mutex
	defValue []byte
}

// Block blocks task by fingeprint for needed TTL.
func (b *Blocker) Block(fingerprint string, ttl time.Duration) (blockedSuccessfully bool, err error) {
	b.mt.Lock()
	defer b.mt.Unlock()

	res, err := b.cache.Get([]byte(fingerprint))
	if len(res) != 0 {
		return false, nil
	}

	if err != nil && !strings.Contains(err.Error(), "not found") {
		return false, err
	}

	err = b.cache.Set([]byte(fingerprint), b.defValue, int(ttl.Seconds()))
	if err != nil {
		return false, err
	}

	return true, nil
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
