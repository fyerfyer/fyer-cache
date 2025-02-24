package distributelock

import "context"

// Lock 定义分布式锁接口
type Lock interface {
	// Lock 获取锁
	// 如果获取失败会阻塞直到获取成功或上下文取消
	Lock(ctx context.Context) error

	// TryLock 尝试获取锁
	// 如果获取失败则立即返回错误
	TryLock(ctx context.Context) error

	// Unlock 释放锁
	Unlock(ctx context.Context) error

	// Refresh 手动续约
	// 将重置锁的过期时间
	Refresh(ctx context.Context) error
}
