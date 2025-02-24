package ferr

import "errors"

var (
	// ErrKeyNotFound 键未找到错误
	ErrKeyNotFound = errors.New("key not found")
)

var (
	// ErrLockNotHeld 未持有锁
	ErrLockNotHeld = errors.New("lock not held")

	// ErrLockAcquireFailed 获取锁失败
	ErrLockAcquireFailed = errors.New("failed to acquire lock")

	// ErrLockAlreadyHeld 锁已被持有
	ErrLockAlreadyHeld = errors.New("lock already held")

	// ErrRefreshFailed 续约失败
	ErrRefreshFailed = errors.New("failed to refresh lock")
)
