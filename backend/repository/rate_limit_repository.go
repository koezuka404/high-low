package repository

import (
	"context"
	"fmt"

	"backend/usecase"

	"github.com/redis/go-redis/v9"
)

type rateLimitRepository struct {
	client redis.UniversalClient
	script *redis.Script
}

const tokenBucketLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local refill_rate = tonumber(ARGV[3])
local token_cost = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])
local tokens_str = redis.call("HGET", key, "tokens")
local last_str = redis.call("HGET", key, "last_refill")
local tokens, last_refill
if tokens_str == false or tokens_str == nil then
  tokens = capacity
  last_refill = now
else
  tokens = tonumber(tokens_str)
  last_refill = tonumber(last_str)
  local elapsed = now - last_refill
  if elapsed > 0 then
    tokens = math.min(capacity, tokens + elapsed * refill_rate)
  end
  last_refill = now
end
if tokens >= token_cost then
  tokens = tokens - token_cost
  redis.call("HSET", key, "tokens", tostring(tokens), "last_refill", tostring(now))
  redis.call("EXPIRE", key, ttl)
  return {1, 0}
else
  local retry_after = math.ceil((token_cost - tokens) / refill_rate)
  if retry_after < 1 then retry_after = 1 end
  return {0, retry_after}
end
`

// NewRateLimitRepository は Redis クライアントが nil のとき nil を返す（noop は使わない）。
func NewRateLimitRepository(client redis.UniversalClient) usecase.RateLimiter {
	if client == nil {
		return nil
	}
	return &rateLimitRepository{
		client: client,
		script: redis.NewScript(tokenBucketLua),
	}
}

func (r *rateLimitRepository) ConsumeToken(ctx context.Context, key string, now float64, capacity, refillRate, tokenCost float64, ttlSec int64) (allowed bool, retryAfterSec int, err error) {
	result, err := r.script.Run(ctx, r.client, []string{key},
		now, capacity, refillRate, tokenCost, ttlSec).Result()
	if err != nil {
		return false, 0, fmt.Errorf("rate limit script: %w", err)
	}
	slice, ok := result.([]interface{})
	if !ok || len(slice) < 2 {
		return false, 1, fmt.Errorf("rate limit unexpected result: %T %v", result, result)
	}
	allowedVal, _ := toInt64(slice[0])
	retryVal, _ := toInt64(slice[1])
	return allowedVal == 1, int(retryVal), nil
}

func toInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	default:
		return 0, false
	}
}
