package redis_lock

import (
	"context"
	"errors"
	"redis_lock/utils"
	"sync/atomic"
	"time"
)

const RedisLockKeyPrefix = "REDIS_LOCK_"

var ErrLockAcquireByOthers = errors.New("lock is acquired by others")

func IsRetryableErr(err error) bool {
	return errors.Is(err, ErrLockAcquireByOthers)
}

type RedisLock struct {
	LockOptions
	key    string
	token  string
	client *MyRedis
	// watch dog state
	runningDog int32
	// stop watcg dog thread
	stopDog context.CancelFunc
}

func NewRedisLock(key string, client *MyRedis, opts ...LockOption) *RedisLock {
	r := RedisLock{
		key:    RedisLockKeyPrefix + key,
		token:  utils.GetProcessAndGoroutineIDStr(),
		client: client,
	}
	for _, opt := range opts {
		opt(&r.LockOptions)
	}
	DefaultLockOptions(&r.LockOptions)
	return &r
}
func (r *RedisLock) Lock(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			// lock err
			return
		}
		// begin watch dog
		r.watchDog(ctx)
	}()
	// try acquire lock
	err = r.tryLock(ctx)
	if err == nil {
		// acquire lock return
		return
	}
	if !IsRetryableErr(err) {
		// error condition
		return
	}
	// block wait block
	err = r.blockWaitLock(ctx)
	return
}
func (r *RedisLock) tryLock(ctx context.Context) (err error) {
	// setNX EX add lock
	result, err := r.client.SetNX(ctx, r.key, r.token, time.Duration(r.expireSeconds)*time.Second).Result()
	if err != nil {
		// run error
		return
	}
	if !result {
		err = ErrLockAcquireByOthers
	}
	return
}
func (r *RedisLock) blockWaitLock(ctx context.Context) (err error) {
	timeoutCh := time.After(time.Duration(r.blockWaitingSeconds) * time.Second)
	ticker := time.NewTicker(DefaultAcquireLockTime)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-timeoutCh:
			// block timeout
			return
		case <-ctx.Done():
			// context timeout
			return
		default:
		}
		err = r.tryLock(ctx)
		if err == nil {
			// acquire lock return
			return
		}
		if !IsRetryableErr(err) {
			// error condition
			return
		}
	}
	// not reach
	return
}
func (r *RedisLock) watchDog(ctx context.Context) {
	// assure runningDog is 0
	for !atomic.CompareAndSwapInt32(&r.runningDog, 0, 1) {
	}
	ctx, r.stopDog = context.WithCancel(ctx)
	go func() {
		defer func() {
			// end lock to change 0
			atomic.StoreInt32(&r.runningDog, 0)
		}()
		r.runWatchDog(ctx)
	}()
}
func (r *RedisLock) runWatchDog(ctx context.Context) {
	ticker := time.NewTicker(WatchDogWorkTime)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := r.delayExpireLock(ctx, r.expireSeconds)
		if err != nil {
			// watch dog expire lock error
		}
	}
}
func (r *RedisLock) delayExpireLock(ctx context.Context, expire int64) error {
	result, err := r.client.Eval(ctx, LuaGetAndExpire, []string{r.key}, r.token, expire).Result()
	if err != nil {
		return err
	}
	if num, _ := result.(int64); num != 1 {
		return errors.New("lock already timeout or not belong to this token")
	}
	return nil
}
func (r *RedisLock) Unlock(ctx context.Context) error {
	defer func() {
		if r.stopDog != nil {
			r.stopDog()
		}
	}()
	result, err := r.client.Eval(ctx, LuaCheckAndDeleteLock, []string{r.key}, r.token).Result()
	if err != nil {
		return err
	}
	if num, _ := result.(int64); num != 1 {
		return errors.New("not unlock timeout or no ownership of lock")
	}
	return nil
}
