package blocker

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	foreverTTL = 0
	defValue   = "l"
)

// Blocker blocks tasks by executor name and fingerprint.
type Blocker struct {
	cache    cacher
	mt       *sync.Mutex
	defValue []byte
}

// BlockInProgress blocks task by executor name and fingeprint while executing.
func (b *Blocker) BlockInProgress(executor, fingerprint string) (blockedSuccessfully bool, err error) {
	b.mt.Lock()
	defer b.mt.Unlock()

	key := getBlockKey(executor, fingerprint)

	res, err := b.cache.Get(key)
	if len(res) != 0 {
		return false, nil
	}

	if err != nil && !strings.Contains(err.Error(), "not found") {
		return false, err
	}

	err = b.cache.Set(key, b.defValue, foreverTTL)
	if err != nil {
		return false, err
	}

	return true, nil
}

// BlockForTTL blocks task by executor name and fingeprint for needed TTL.
func (b *Blocker) BlockForTTL(executor, fingerprint string, ttl time.Duration) error {
	b.mt.Lock()
	defer b.mt.Unlock()

	return b.cache.Set(getBlockKey(executor, fingerprint), b.defValue, int(ttl.Seconds()))
}

// Unblock unblocks task by executor name and fingeprint.
func (b *Blocker) Unblock(executor, fingerprint string) {
	b.mt.Lock()
	defer b.mt.Unlock()

	_ = b.cache.Del(getBlockKey(executor, fingerprint))
}

// New creates Blocker instance.
func New(cache cacher) *Blocker {
	return &Blocker{
		cache:    cache,
		mt:       &sync.Mutex{},
		defValue: []byte(defValue),
	}
}

func getBlockKey(executor, fingerprint string) []byte {
	return []byte(fmt.Sprintf("%v_%v", executor, fingerprint))
}

//go:generate mockgen -source=blocker.go -destination=blocker_mocks.go -package=blocker doc github.com/golang/mock/gomock

type cacher interface {
	Get(key []byte) (value []byte, err error)
	Set(key, value []byte, expireSeconds int) (err error)
	Del(key []byte) (affected bool)
}
