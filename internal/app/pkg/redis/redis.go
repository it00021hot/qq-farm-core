package redis

import (
	"context"
	"errors"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/vars"
	"github.com/it00021hot/qq-farm-core/pkg/memcache"
)

// ErrNil mimics redis.Nil when key is missing.
var ErrNil = errors.New("memcache: nil")

const (
	AuthFmt  = "auth:%s"   // 授权信息key
	PermsFmt = "perms:%s"  // 用户详细信息 (基于用户的菜单权限 包含用户前端权限 alias)
	ResFmt   = "resources" // 资源列表
)

func prefix() string {
	p := vars.Config.GetString("redis.prefix")
	if p == "" {
		p = vars.Config.GetString("cache.prefix")
	}
	return p
}

func BuildAuthRedisKey(key string) string {
	return prefix() + key
}

// ---- compatible command wrappers (Result / Err) ----

type StatusCmd struct{ err error }

func (c *StatusCmd) Err() error { return c.err }

type StringCmd struct {
	val string
	err error
}

func (c *StringCmd) Result() (string, error) { return c.val, c.err }
func (c *StringCmd) Err() error              { return c.err }

type IntCmd struct {
	val int64
	err error
}

func (c *IntCmd) Result() (int64, error) { return c.val, c.err }
func (c *IntCmd) Err() error             { return c.err }

type StringSliceCmd struct {
	val []string
	err error
}

func (c *StringSliceCmd) Result() ([]string, error) { return c.val, c.err }
func (c *StringSliceCmd) Err() error                { return c.err }

type BoolCmd struct {
	val bool
	err error
}

func (c *BoolCmd) Result() (bool, error) { return c.val, c.err }
func (c *BoolCmd) Err() error            { return c.err }

type DurationCmd struct {
	val time.Duration
	err error
}

func (c *DurationCmd) Result() (time.Duration, error) { return c.val, c.err }
func (c *DurationCmd) Err() error                     { return c.err }

type MapStringStringCmd struct {
	val map[string]string
	err error
}

func (c *MapStringStringCmd) Result() (map[string]string, error) { return c.val, c.err }
func (c *MapStringStringCmd) Err() error                         { return c.err }

func Keys(ctx context.Context, key string) *StringSliceCmd {
	_ = ctx
	rKey := BuildAuthRedisKey(key)
	return &StringSliceCmd{val: memcache.Default.Keys(rKey)}
}

func Set(ctx context.Context, key string, value string) *StatusCmd {
	_ = ctx
	ttl := vars.Config.GetDuration("jwt.expire") * time.Second
	memcache.Default.Set(BuildAuthRedisKey(key), value, ttl)
	return &StatusCmd{}
}

func SetEx(ctx context.Context, key string, value string, expireAt time.Duration) *StatusCmd {
	_ = ctx
	memcache.Default.Set(BuildAuthRedisKey(key), value, expireAt)
	return &StatusCmd{}
}

func SetNX(ctx context.Context, key string, value string) *BoolCmd {
	_ = ctx
	rKey := BuildAuthRedisKey(key)
	if _, ok := memcache.Default.Get(rKey); ok {
		return &BoolCmd{val: false}
	}
	memcache.Default.Set(rKey, value, 0)
	return &BoolCmd{val: true}
}

func Del(ctx context.Context, key string) *IntCmd {
	_ = ctx
	n := memcache.Default.Del(BuildAuthRedisKey(key))
	return &IntCmd{val: n}
}

func DelRaw(ctx context.Context, keys ...string) *IntCmd {
	_ = ctx
	if len(keys) == 0 {
		return &IntCmd{val: 0}
	}
	return &IntCmd{val: memcache.Default.Del(keys...)}
}

// InvalidatePermsCache 清除全部用户前端权限缓存
func InvalidatePermsCache(ctx context.Context) {
	keys, err := Keys(ctx, "perms:*").Result()
	if err != nil || len(keys) == 0 {
		_ = Del(ctx, ResFmt).Err()
		return
	}
	_ = DelRaw(ctx, keys...).Err()
	_ = Del(ctx, ResFmt).Err()
}

func Get(ctx context.Context, key string) *StringCmd {
	_ = ctx
	val, ok := memcache.Default.Get(BuildAuthRedisKey(key))
	if !ok {
		return &StringCmd{err: ErrNil}
	}
	return &StringCmd{val: val}
}

func Exists(ctx context.Context, key string) *IntCmd {
	_ = ctx
	if _, ok := memcache.Default.Get(BuildAuthRedisKey(key)); ok {
		return &IntCmd{val: 1}
	}
	return &IntCmd{val: 0}
}

func LRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	_ = ctx
	_ = key
	_ = start
	_ = stop
	return &StringSliceCmd{val: nil}
}

func LRem(ctx context.Context, key string, count int64, value interface{}) *IntCmd {
	_ = ctx
	_ = key
	_ = count
	_ = value
	return &IntCmd{val: 0}
}

func LPush(ctx context.Context, key string, values ...interface{}) *IntCmd {
	_ = ctx
	_ = key
	_ = values
	return &IntCmd{val: 0}
}

func TTL(ctx context.Context, key string) *DurationCmd {
	_ = ctx
	_ = key
	return &DurationCmd{val: -1}
}

func GetSet(ctx context.Context, key string, value interface{}) *StringCmd {
	_ = ctx
	rKey := BuildAuthRedisKey(key)
	old, _ := memcache.Default.Get(rKey)
	memcache.Default.Set(rKey, toString(value), 0)
	return &StringCmd{val: old}
}

func HMSet(ctx context.Context, key string, values ...interface{}) *BoolCmd {
	_ = ctx
	_ = key
	_ = values
	return &BoolCmd{val: true}
}

func HGetAll(ctx context.Context, key string) *MapStringStringCmd {
	_ = ctx
	_ = key
	return &MapStringStringCmd{val: map[string]string{}}
}

func HGet(ctx context.Context, key string, field string) *StringCmd {
	_ = ctx
	_ = key
	_ = field
	return &StringCmd{err: ErrNil}
}

func SMembers(ctx context.Context, key string) *StringSliceCmd {
	_ = ctx
	_ = key
	return &StringSliceCmd{val: nil}
}

func SRem(ctx context.Context, key string, values ...interface{}) *IntCmd {
	_ = ctx
	_ = key
	_ = values
	return &IntCmd{val: 0}
}

func Publish(ctx context.Context, topic, msg string) error {
	_ = ctx
	_ = topic
	_ = msg
	return nil
}

func Subscribe(ctx context.Context, topic string) interface{} {
	_ = ctx
	_ = topic
	return nil
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}
