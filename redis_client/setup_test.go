package redis_client

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	redis "github.com/go-redis/redis/v8"
)

func TestRedisConfigLogFields_RedactsPassword(t *testing.T) {
	config := Config{
		Host:     "127.0.0.1",
		Port:     "6379",
		Password: "super-secret",
		DB:       2,
	}

	logFields := redisConfigLogFields(config)
	if strings.Contains(logFields, config.Password) {
		t.Fatalf("log fields leak password: %s", logFields)
	}
	if !strings.Contains(logFields, "password=[REDACTED]") {
		t.Fatalf("log fields should contain redaction marker, got: %s", logFields)
	}
}

func TestRedisConfigLogFields_EmptyPassword(t *testing.T) {
	config := Config{
		Host: "127.0.0.1",
		Port: "6379",
		DB:   0,
	}

	logFields := redisConfigLogFields(config)
	if !strings.Contains(logFields, "password=<empty>") {
		t.Fatalf("log fields should mark empty password, got: %s", logFields)
	}
}

func integrationRedisConfig(t *testing.T) Config {
	t.Helper()

	addr := strings.TrimSpace(os.Getenv("REDIS_TEST_ADDR"))
	if addr == "" {
		t.Skip("set REDIS_TEST_ADDR to run redis integration tests")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("invalid REDIS_TEST_ADDR %q: %v", addr, err)
	}

	db := 0
	if dbRaw := strings.TrimSpace(os.Getenv("REDIS_TEST_DB")); dbRaw != "" {
		parsed, parseErr := strconv.Atoi(dbRaw)
		if parseErr != nil {
			t.Fatalf("invalid REDIS_TEST_DB %q: %v", dbRaw, parseErr)
		}
		db = parsed
	}

	return Config{
		Host:     host,
		Port:     port,
		Password: os.Getenv("REDIS_TEST_PASSWORD"),
		DB:       db,
	}
}

func TestNewRedis_ConnectionSuccess(t *testing.T) {
	config := integrationRedisConfig(t)

	client, err := NewRedis(config)
	if err != nil {
		t.Fatalf("NewRedis() failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}

	if pong != "PONG" {
		t.Errorf("Ping() returned %v, want PONG", pong)
	}
}

func TestNewRedis_ConnectionFailure_UnreachablePort(t *testing.T) {
	config := Config{
		Host: "127.0.0.1",
		Port: "1",
		DB:   0,
	}

	_, err := NewRedis(config)
	if err == nil {
		t.Fatal("NewRedis() should fail when port is unreachable")
	}
}

func TestNewRedis_ConnectionFailure_WrongHost(t *testing.T) {
	config := Config{
		Host: "nonexistent-host.invalid",
		Port: "6379",
		DB:   0,
	}

	_, err := NewRedis(config)
	if err == nil {
		t.Fatal("NewRedis() should fail with wrong host")
	}
}

func TestNewRedis_Operations(t *testing.T) {
	config := integrationRedisConfig(t)

	client, err := NewRedis(config)
	if err != nil {
		t.Fatalf("NewRedis() failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testKey := "test_key"
	testValue := "test_value"
	err = client.Set(ctx, testKey, testValue, 0).Err()
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	val, err := client.Get(ctx, testKey).Result()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if val != testValue {
		t.Errorf("Get() returned %v, want %v", val, testValue)
	}

	err = client.Del(ctx, testKey).Err()
	if err != nil {
		t.Fatalf("Del() failed: %v", err)
	}

	_, err = client.Get(ctx, testKey).Result()
	if err == nil {
		t.Error("Key should be deleted")
	}
}

func TestNewRedis_ConcurrentConnections(t *testing.T) {
	config := integrationRedisConfig(t)

	clients := make([]*redis.Client, 5)
	for i := 0; i < 5; i++ {
		client, err := NewRedis(config)
		if err != nil {
			t.Fatalf("NewRedis() failed for client %d: %v", i, err)
		}
		clients[i] = client
	}

	for _, client := range clients {
		if client != nil {
			client.Close()
		}
	}
}
