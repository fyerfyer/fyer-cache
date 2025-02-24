package distributelock

import (
	"context"
	"testing"

	"github.com/fyerfyer/fyer-cache/internal/ferr"
	"github.com/fyerfyer/fyer-cache/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"time"
)

// TestRedisLock_Lock 测试RedisLock的加锁功能
func TestRedisLock_Lock(t *testing.T) {
	testCases := []struct {
		name      string
		mock      func(mck *mocks.Cmdable)
		wantError error
	}{
		{
			name: "lock success",
			mock: func(mock *mocks.Cmdable) {
				cmd := redis.NewCmd(context.Background())
				cmd.SetVal(int64(1))
				mock.On("Eval",
					mock2.Anything,
					lockScript,
					[]string{"test-key"},
					mock2.Anything, // lockVal
					int64(30000),   // expireMS
				).Return(cmd)
			},
			wantError: nil,
		},
		{
			name: "lock held by others",
			mock: func(mock *mocks.Cmdable) {
				cmd := redis.NewCmd(context.Background())
				cmd.SetVal(int64(0)) // 修改返回值为int64(0)
				mock.On("Eval",
					mock2.Anything,
					lockScript,
					[]string{"test-key"},
					mock2.Anything, // lockVal
					int64(30000),   // expireMS
				).Return(cmd)
			},
			wantError: ferr.ErrLockAcquireFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRedis := mocks.NewCmdable(t)
			tc.mock(mockRedis)

			lock := NewRedisLock(mockRedis, "test-key",
				WithExpiration(30*time.Second),
				WithBlockWaiting(false))
			err := lock.TryLock(context.Background())
			assert.Equal(t, tc.wantError, err)
		})
	}
}

// TestRedisLock_Unlock 测试RedisLock的解锁功能
func TestRedisLock_Unlock(t *testing.T) {
	testCases := []struct {
		name      string
		mock      func(mock *mocks.Cmdable, lockVal string)
		wantError error
	}{
		{
			name: "unlock success",
			mock: func(mock *mocks.Cmdable, lockVal string) {
				// 先模拟加锁成功
				cmdLock := redis.NewCmd(context.Background())
				cmdLock.SetVal("OK")
				mock.On("Eval",
					context.Background(),
					lockScript,
					[]string{"test-key"},
					[]interface{}{lockVal, int64(30000)}).
					Return(cmdLock)

				// 再模拟解锁成功
				cmdUnlock := redis.NewCmd(context.Background())
				cmdUnlock.SetVal(int64(1))
				mock.On("Eval",
					context.Background(),
					unlockScript,
					[]string{"test-key"},
					[]interface{}{lockVal}).
					Return(cmdUnlock)
			},
			wantError: nil,
		},
		{
			name: "unlock not held lock",
			mock: func(mock *mocks.Cmdable, lockVal string) {
				cmdUnlock := redis.NewCmd(context.Background())
				cmdUnlock.SetVal(int64(0))
				mock.On("Eval",
					context.Background(),
					unlockScript,
					[]string{"test-key"},
					[]interface{}{lockVal}).
					Return(cmdUnlock)
			},
			wantError: ferr.ErrLockNotHeld,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRedis := mocks.NewCmdable(t)
			lock := NewRedisLock(mockRedis, "test-key",
				WithExpiration(30*time.Second),
				WithWatchdog(false))

			if tc.name == "unlock success" {
				err := lock.TryLock(context.Background())
				require.NoError(t, err)
			}

			tc.mock(mockRedis, lock.val)
			err := lock.Unlock(context.Background())
			assert.Equal(t, tc.wantError, err)
		})
	}
}

// TestRedisLock_Refresh 测试RedisLock的续约功能
func TestRedisLock_Refresh(t *testing.T) {
	testCases := []struct {
		name      string
		mock      func(mock *mocks.Cmdable, lockVal string)
		wantError error
	}{
		{
			name: "refresh success",
			mock: func(mock *mocks.Cmdable, lockVal string) {
				// 先模拟加锁成功
				cmdLock := redis.NewCmd(context.Background())
				cmdLock.SetVal("OK")
				mock.On("Eval",
					context.Background(),
					lockScript,
					[]string{"test-key"},
					[]interface{}{lockVal, int64(30000)}).
					Return(cmdLock)

				// 再模拟续约成功
				cmdRefresh := redis.NewCmd(context.Background())
				cmdRefresh.SetVal(int64(1))
				mock.On("Eval",
					context.Background(),
					refreshScript,
					[]string{"test-key"},
					[]interface{}{lockVal, int64(30000)}).
					Return(cmdRefresh)
			},
			wantError: nil,
		},
		{
			name: "refresh expired lock",
			mock: func(mock *mocks.Cmdable, lockVal string) {
				cmdRefresh := redis.NewCmd(context.Background())
				cmdRefresh.SetVal(int64(0))
				mock.On("Eval",
					context.Background(),
					refreshScript,
					[]string{"test-key"},
					[]interface{}{lockVal, int64(30000)}).
					Return(cmdRefresh)
			},
			wantError: ferr.ErrLockNotHeld,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRedis := mocks.NewCmdable(t)
			lock := NewRedisLock(mockRedis, "test-key",
				WithExpiration(30*time.Second),
				WithWatchdog(false))

			if tc.name == "refresh success" {
				err := lock.TryLock(context.Background())
				require.NoError(t, err)
			}

			tc.mock(mockRedis, lock.val)
			err := lock.Refresh(context.Background())
			assert.Equal(t, tc.wantError, err)
		})
	}
}
