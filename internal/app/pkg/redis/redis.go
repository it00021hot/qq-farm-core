package redis

import (
	"context"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/vars"
	"github.com/redis/go-redis/v9"
)

const (
	AuthFmt  = "auth:%s"   // 授权信息key
	PermsFmt = "perms:%s"  // 用户详细信息 (基于用户的菜单权限 包含用户前端权限 alias)
	ResFmt   = "resources" // 资源列表
)

func BuildAuthRedisKey(key string) string {
	return vars.Config.GetString("redis.prefix") + key
}

func Keys(ctx context.Context, key string) *redis.StringSliceCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.Keys(ctx, rKey)
}

func Set(ctx context.Context, key string, value string) *redis.StatusCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.Set(ctx, rKey, value, vars.Config.GetDuration("jwt.expire")*time.Second)
}

func SetEx(ctx context.Context, key string, value string, expireAt time.Duration) *redis.StatusCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.Set(ctx, rKey, value, expireAt)
}

// SetNX 设置key 如果不存在 则设置 过期时间 返回bool值 true 表示设置成功 false 表示设置失败 幂等性操作
func SetNX(ctx context.Context, key string, value string) *redis.BoolCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.SetNX(ctx, rKey, value, 0)
}

func Del(ctx context.Context, key string) *redis.IntCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.Del(ctx, rKey)
}

func Get(ctx context.Context, key string) *redis.StringCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.Get(ctx, rKey)
}

func Exists(ctx context.Context, key string) *redis.IntCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.Exists(ctx, rKey)
}

func LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.LRange(ctx, rKey, start, stop)
}

func LRem(ctx context.Context, key string, count int64, value interface{}) *redis.IntCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.LRem(ctx, rKey, count, value)
}

func LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.LPush(ctx, rKey, values)
}

// TTL 获取key过期时间
func TTL(ctx context.Context, key string) *redis.DurationCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.TTL(ctx, rKey)
}

// GetSet 获取key的值 并设置新的值
func GetSet(ctx context.Context, key string, value interface{}) *redis.StringCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.GetSet(ctx, rKey, value)
}

// HMSet 设置hash的值
func HMSet(ctx context.Context, key string, values ...interface{}) *redis.BoolCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.HMSet(ctx, rKey, values)
}

// HGetAll 获取hash的值
func HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.HGetAll(ctx, rKey)
}

// HGet 获取hash的值
func HGet(ctx context.Context, key string, field string) *redis.StringCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.HGet(ctx, rKey, field)
}

// SMembers 获取集合的值
func SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.SMembers(ctx, rKey)
}

// SRem 删除集合的值
func SRem(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	rKey := BuildAuthRedisKey(key)
	return vars.Redis.SRem(ctx, rKey, values)
}

// Publish 发布消息到 Redis topic
func Publish(ctx context.Context, topic, msg string) error {
	return vars.Redis.Publish(ctx, topic, msg).Err()
}

// Subscribe 订阅 Redis topic，返回 *redis.PubSub
func Subscribe(ctx context.Context, topic string) *redis.PubSub {
	return vars.Redis.Subscribe(ctx, topic)
}
