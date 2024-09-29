package redis_lock

import "time"

const (
	DefaultBlockWaitingSeconds = 10
	DefaultLockExpireSeconds   = 30
	DefaultAcquireLockTime     = time.Duration(5) * time.Millisecond
	WatchDogWorkTime           = time.Duration(DefaultLockExpireSeconds/3) * time.Second
)

type LockOption func(*LockOptions)

type LockOptions struct {
	blockWaitingSeconds int64
	expireSeconds       int64
}

func WithBlockWaitingSeconds(waitingSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.blockWaitingSeconds = waitingSeconds
	}
}
func WithExpireSeconds(expire int64) LockOption {
	return func(o *LockOptions) {
		o.expireSeconds = expire
	}
}
func DefaultLockOptions(o *LockOptions) {
	if o.blockWaitingSeconds <= 0 {
		o.blockWaitingSeconds = DefaultBlockWaitingSeconds
	}
	if o.expireSeconds <= 0 {
		o.expireSeconds = DefaultLockExpireSeconds
	}
}
