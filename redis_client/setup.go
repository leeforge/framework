package redis_client

import (
	"context"
	"fmt"

	redis "github.com/go-redis/redis/v8"
	"github.com/leeforge/framework/env_mode"
)

func NewRedis(cnf Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cnf.Addr(),
		Password: cnf.Password,
		DB:       cnf.DB,
	})
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	if env_mode.Mode() == env_mode.DevMode {
		fmt.Printf("redis连接成功: %s (%s)\n", pong, redisConfigLogFields(cnf))
	}
	return client, nil
}

func redisConfigLogFields(cnf Config) string {
	return fmt.Sprintf("addr=%s db=%d password=%s", cnf.Addr(), cnf.DB, redactedPassword(cnf.Password))
}

func redactedPassword(password string) string {
	if password == "" {
		return "<empty>"
	}
	return "[REDACTED]"
}
