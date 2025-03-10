package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultCleanerInterval 默认清理间隔
const DefaultCleanerInterval = 2 * time.Minute

// DefaultWorkerCount 默认清理工作协程数量
const DefaultWorkerCount = 4

// DefaultQueueSize 默认清理任务队列大小
const DefaultQueueSize = 128

// CleanTask 清理任务
type CleanTask struct {
	shardIndex int             // 要清理的分片索引
	now        time.Time       // 清理时的当前时间
	ctx        context.Context // 上下文，用于取消任务
}

// Cleaner 清理器结构
// 负责异步清理过期的缓存项
type Cleaner struct {
	cache       *ShardedMap       // 要清理的分片缓存
	taskQueue   chan CleanTask    // 任务队列
	workerCount int               // 工作协程数量
	interval    time.Duration     // 清理周期
	stopChan    chan struct{}     // 停止信号
	wg          sync.WaitGroup    // 用于等待所有worker完成
	onEvict     func(string, any) // 淘汰回调
	running     bool              // 是否已启动
	mu          sync.Mutex        // 保护running状态
}

// NewCleaner 创建清理器
func NewCleaner(cache *ShardedMap, options ...CleanerOption) *Cleaner {
	cleaner := &Cleaner{
		cache:       cache,
		interval:    DefaultCleanerInterval,
		workerCount: DefaultWorkerCount,
		stopChan:    make(chan struct{}),
	}

	// 应用选项
	for _, option := range options {
		option(cleaner)
	}

	// 初始化任务队列
	if cleaner.taskQueue == nil {
		cleaner.taskQueue = make(chan CleanTask, DefaultQueueSize)
	}

	return cleaner
}

// WithCleanInterval 设置清理间隔
func WithCleanInterval(interval time.Duration) CleanerOption {
	return func(c *Cleaner) {
		if interval > 0 {
			c.interval = interval
		}
	}
}

// WithQueueSize 设置队列大小
func WithQueueSize(size int) CleanerOption {
	return func(c *Cleaner) {
		if size > 0 {
			c.taskQueue = make(chan CleanTask, size)
		}
	}
}

// CleanerWithEvictionCallback 设置淘汰回调
func CleanerWithEvictionCallback(callback func(key string, value any)) CleanerOption {
	return func(c *Cleaner) {
		c.onEvict = callback
	}
}

// Start 启动清理器
func (c *Cleaner) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return
	}

	c.running = true

	// 启动工作协程池
	for i := 0; i < c.workerCount; i++ {
		c.wg.Add(1)
		go c.worker()
	}

	// 启动调度器
	go c.scheduler()
}

// Stop 停止清理器
func (c *Cleaner) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	// 发送停止信号
	close(c.stopChan)

	// 等待所有工作协程结束
	c.wg.Wait()
}

// worker 工作协程，处理清理任务
func (c *Cleaner) worker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopChan:
			return // 收到停止信号，退出
		case task := <-c.taskQueue:
			c.processTask(task)
		}
	}
}

// scheduler 调度器，定期安排清理任务
func (c *Cleaner) scheduler() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return // 收到停止信号，退出
		case <-ticker.C:
			c.scheduleCleanTasks()
		}
	}
}

// scheduleCleanTasks 安排清理任务
func (c *Cleaner) scheduleCleanTasks() {
	now := time.Now()
	ctx := context.Background()

	// 为每个分片创建一个清理任务
	for i := 0; i < c.cache.ShardCount(); i++ {
		select {
		case <-c.stopChan:
			return // 如果收到停止信号，立即退出
		case c.taskQueue <- CleanTask{
			shardIndex: i,
			now:        now,
			ctx:        ctx,
		}:
			// 成功将任务加入队列
		}
	}
}

// processTask 处理单个清理任务
func (c *Cleaner) processTask(task CleanTask) {
	// 使用 RangeShard 遍历指定分片的所有缓存项
	c.cache.RangeShard(task.shardIndex, func(key string, value *cacheItem) bool {
		// 检查是否过期
		if !value.expiration.IsZero() && task.now.After(value.expiration) {
			// 获取分片
			shard := c.cache.shards[task.shardIndex]

			// 锁定分片
			shard.mu.Lock()

			// 再次检查项目是否存在并且已过期
			// 这是必要的，因为在获取锁的过程中可能已经被修改
			if item, exists := shard.items[key]; exists && !item.expiration.IsZero() && task.now.After(item.expiration) {
				// 删除过期项目
				delete(shard.items, key)
				// 更新计数
				atomic.AddInt64(&c.cache.itemCount, -1)

				// 解锁分片，以便后续操作
				shard.mu.Unlock()

				// 调用淘汰回调
				if c.onEvict != nil {
					c.onEvict(key, item.value)
				}
			} else {
				shard.mu.Unlock()
			}
		}

		// 检查任务是否已取消
		select {
		case <-task.ctx.Done():
			return false // 停止遍历
		default:
			return true // 继续遍历
		}
	})
}

// ScheduleManualClean 手动安排清理任务
// 可在任何时候调用，用于触发立即清理
func (c *Cleaner) ScheduleManualClean(ctx context.Context) {
	now := time.Now()

	// 为每个分片创建一个清理任务
	for i := 0; i < c.cache.ShardCount(); i++ {
		select {
		case <-c.stopChan:
			return // 如果收到停止信号，立即退出
		case <-ctx.Done():
			return // 如果上下文取消，立即退出
		case c.taskQueue <- CleanTask{
			shardIndex: i,
			now:        now,
			ctx:        ctx,
		}:
			// 成功将任务加入队列
		}
	}
}
