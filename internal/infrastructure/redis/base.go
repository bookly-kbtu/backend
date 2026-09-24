package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/bookly-kbtu/backend/internal/domain"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type Client struct {
	*goredis.Client
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{Client: client}, nil
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}

	if err := c.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set cache key %q: %w", key, err)
	}

	return nil
}

// GetJSON returns domain.ErrNotFound when the key does not exist.
func (c *Client) GetJSON(ctx context.Context, key string, dest any) error {
	data, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("get cache key %q: %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal cache value: %w", err)
	}

	return nil
}

// Incr increments a counter and sets ttl on first hit. Used for OTP/IP rate limits.
func (c *Client) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("incr key %q: %w", key, err)
	}

	return incr.Val(), nil
}

// Allow implements domain.RateLimiter with a fixed window counter.
func (c *Client) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	count, err := c.Incr(ctx, "ratelimit:"+key, window)
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}
