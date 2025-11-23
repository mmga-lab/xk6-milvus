package milvus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.False(t, config.Debug)
	assert.Empty(t, config.Address)
	assert.Empty(t, config.Username)
	assert.Empty(t, config.Password)
	assert.Empty(t, config.DefaultCollection)
}

func TestWithAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "localhost with port",
			address: "localhost:19530",
		},
		{
			name:    "ip address",
			address: "192.168.1.100:19530",
		},
		{
			name:    "domain name",
			address: "milvus.example.com:19530",
		},
		{
			name:    "empty address",
			address: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			option := WithAddress(tt.address)
			option(config)

			assert.Equal(t, tt.address, config.Address)
		})
	}
}

func TestWithAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "valid credentials",
			username: "admin",
			password: "password123",
		},
		{
			name:     "empty credentials",
			username: "",
			password: "",
		},
		{
			name:     "username only",
			username: "user",
			password: "",
		},
		{
			name:     "password only",
			username: "",
			password: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			option := WithAuth(tt.username, tt.password)
			option(config)

			assert.Equal(t, tt.username, config.Username)
			assert.Equal(t, tt.password, config.Password)
		})
	}
}

func TestWithCollection(t *testing.T) {
	tests := []struct {
		name       string
		collection string
	}{
		{
			name:       "valid collection name",
			collection: "my_collection",
		},
		{
			name:       "collection with underscores",
			collection: "test_collection_123",
		},
		{
			name:       "empty collection",
			collection: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			option := WithCollection(tt.collection)
			option(config)

			assert.Equal(t, tt.collection, config.DefaultCollection)
		})
	}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "5 seconds",
			timeout: 5 * time.Second,
		},
		{
			name:    "1 minute",
			timeout: 1 * time.Minute,
		},
		{
			name:    "zero timeout",
			timeout: 0,
		},
		{
			name:    "negative timeout",
			timeout: -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			option := WithTimeout(tt.timeout)
			option(config)

			assert.Equal(t, tt.timeout, config.Timeout)
		})
	}
}

func TestWithMaxRetries(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
	}{
		{
			name:       "zero retries",
			maxRetries: 0,
		},
		{
			name:       "positive retries",
			maxRetries: 5,
		},
		{
			name:       "negative retries",
			maxRetries: -1,
		},
		{
			name:       "large number of retries",
			maxRetries: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			option := WithMaxRetries(tt.maxRetries)
			option(config)

			assert.Equal(t, tt.maxRetries, config.MaxRetries)
		})
	}
}

func TestWithDebug(t *testing.T) {
	tests := []struct {
		name  string
		debug bool
	}{
		{
			name:  "enable debug",
			debug: true,
		},
		{
			name:  "disable debug",
			debug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			option := WithDebug(tt.debug)
			option(config)

			assert.Equal(t, tt.debug, config.Debug)
		})
	}
}

func TestApplyOptions(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		config := DefaultClientConfig()
		originalTimeout := config.Timeout
		originalRetries := config.MaxRetries

		config.ApplyOptions()

		assert.Equal(t, originalTimeout, config.Timeout)
		assert.Equal(t, originalRetries, config.MaxRetries)
	})

	t.Run("single option", func(t *testing.T) {
		config := DefaultClientConfig()
		config.ApplyOptions(WithAddress("localhost:19530"))

		assert.Equal(t, "localhost:19530", config.Address)
		assert.Equal(t, 30*time.Second, config.Timeout) // default unchanged
	})

	t.Run("multiple options", func(t *testing.T) {
		config := DefaultClientConfig()
		config.ApplyOptions(
			WithAddress("localhost:19530"),
			WithAuth("admin", "password"),
			WithCollection("test_collection"),
			WithTimeout(60*time.Second),
			WithMaxRetries(5),
			WithDebug(true),
		)

		assert.Equal(t, "localhost:19530", config.Address)
		assert.Equal(t, "admin", config.Username)
		assert.Equal(t, "password", config.Password)
		assert.Equal(t, "test_collection", config.DefaultCollection)
		assert.Equal(t, 60*time.Second, config.Timeout)
		assert.Equal(t, 5, config.MaxRetries)
		assert.True(t, config.Debug)
	})

	t.Run("overwriting options", func(t *testing.T) {
		config := DefaultClientConfig()
		config.ApplyOptions(
			WithAddress("first:19530"),
			WithAddress("second:19530"),
		)

		assert.Equal(t, "second:19530", config.Address)
	})

	t.Run("nil options slice", func(t *testing.T) {
		config := DefaultClientConfig()
		originalTimeout := config.Timeout

		config.ApplyOptions(nil...)

		assert.Equal(t, originalTimeout, config.Timeout)
	})
}

func TestClientConfigChaining(t *testing.T) {
	// Test that options can be created and applied in a fluent style
	config := DefaultClientConfig()

	opts := []ClientOption{
		WithAddress("localhost:19530"),
		WithAuth("user", "pass"),
		WithCollection("my_collection"),
	}

	config.ApplyOptions(opts...)

	assert.Equal(t, "localhost:19530", config.Address)
	assert.Equal(t, "user", config.Username)
	assert.Equal(t, "pass", config.Password)
	assert.Equal(t, "my_collection", config.DefaultCollection)
}
