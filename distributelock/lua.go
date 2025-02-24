package distributelock

const lockScript = `
if redis.call('EXISTS', KEYS[1]) == 0 then
    return redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
end
return nil
`

// 释放锁的Lua脚本
const unlockScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
`

// 续约锁的Lua脚本
const refreshScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('PEXPIRE', KEYS[1], ARGV[2])
else
    return 0
end
`
