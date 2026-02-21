package redis_client

import (
	"testing"
)

func TestConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "standard localhost config",
			config: Config{
				Host:     "localhost",
				Port:     "16379",
				Password: "123456",
				DB:       0,
			},
			expected: "localhost:16379",
		},
		{
			name: "custom host and port",
			config: Config{
				Host:     "redis.example.com",
				Port:     "6380",
				Password: "secret",
				DB:       1,
			},
			expected: "redis.example.com:6380",
		},
		{
			name: "IPv4 address",
			config: Config{
				Host:     "192.168.1.100",
				Port:     "6379",
				Password: "",
				DB:       0,
			},
			expected: "192.168.1.100:6379",
		},
		{
			name: "IPv6 address",
			config: Config{
				Host:     "::1",
				Port:     "6379",
				Password: "",
				DB:       0,
			},
			expected: "::1:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Addr()
			if result != tt.expected {
				t.Errorf("Config.Addr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_StructFields(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "16379",
		Password: "123456",
		DB:       0,
	}

	if config.Host != "localhost" {
		t.Errorf("Config.Host = %v, want %v", config.Host, "localhost")
	}

	if config.Port != "16379" {
		t.Errorf("Config.Port = %v, want %v", config.Port, "16379")
	}

	if config.Password != "123456" {
		t.Errorf("Config.Password = %v, want %v", config.Password, "123456")
	}

	if config.DB != 0 {
		t.Errorf("Config.DB = %v, want %v", config.DB, 0)
	}
}

func TestConfig_EmptyPassword(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
	}

	addr := config.Addr()
	if addr != "localhost:6379" {
		t.Errorf("Config.Addr() with empty password = %v, want %v", addr, "localhost:6379")
	}
}

func TestConfig_DifferentDB(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "6379",
		Password: "password",
		DB:       5,
	}

	if config.DB != 5 {
		t.Errorf("Config.DB = %v, want %v", config.DB, 5)
	}
}
