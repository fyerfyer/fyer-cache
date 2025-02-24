package distributelock

import (
	"context"
	"sync/atomic"
	"time"
)

var (
	defaultInterval       = 10 * time.Second
	defaultRefreshTimeout = 3 * time.Second
)

// WatchDog 监视狗结构体定义
type WatchDog struct {
	lock      Lock          // 需要续约的锁
	interval  time.Duration // 续约间隔
	isRunning atomic.Bool   // 运行状态标志
	stopChan  chan struct{} // 停止信号
}

type WatchDogOpt = func(dog *WatchDog)

func WithInterval(interval time.Duration) WatchDogOpt {
	return func(dog *WatchDog) {
		dog.interval = interval
	}
}

// NewWatchDog 创建WatchDog实例
func NewWatchDog(lock Lock) *WatchDog {
	return &WatchDog{
		lock:     lock,
		stopChan: make(chan struct{}),
		interval: defaultInterval,
	}
}

// Start 启动watchdog
func (w *WatchDog) Start() {
	if !w.isRunning.CompareAndSwap(false, true) {
		return // 已经在运行
	}

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopChan:
				return
			case <-ticker.C:
				// 执行续约
				ctx, cancel := context.WithTimeout(context.Background(), defaultRefreshTimeout)
				err := w.lock.Refresh(ctx)
				cancel()

				// 续约失败，停止watchdog
				if err != nil {
					w.Stop()
					return
				}
			}
		}
	}()
}

// Stop 停止watchdog
func (w *WatchDog) Stop() {
	if w.isRunning.CompareAndSwap(true, false) {
		close(w.stopChan)
	}
}
