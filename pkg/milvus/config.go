package milvus

import (
	"time"
)

// ClientConfig represents configuration options for Milvus client
type ClientConfig struct {
	Address           string
	Username          string
	Password          string
	DefaultCollection string
	Timeout           time.Duration
	MaxRetries        int
	Debug             bool
}

// ClientOption is a function that modifies ClientConfig
type ClientOption func(*ClientConfig)

// DefaultClientConfig returns a ClientConfig with default values
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Debug:      false,
	}
}

// WithAddress sets the Milvus server address
func WithAddress(address string) ClientOption {
	return func(c *ClientConfig) {
		c.Address = address
	}
}

// WithAuth sets the authentication credentials
func WithAuth(username, password string) ClientOption {
	return func(c *ClientConfig) {
		c.Username = username
		c.Password = password
	}
}

// WithCollection sets the default collection
func WithCollection(collection string) ClientOption {
	return func(c *ClientConfig) {
		c.DefaultCollection = collection
	}
}

// WithTimeout sets the operation timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(retries int) ClientOption {
	return func(c *ClientConfig) {
		c.MaxRetries = retries
	}
}

// WithDebug enables debug mode
func WithDebug(debug bool) ClientOption {
	return func(c *ClientConfig) {
		c.Debug = debug
	}
}

// ApplyOptions applies a list of options to the config
func (c *ClientConfig) ApplyOptions(opts ...ClientOption) {
	for _, opt := range opts {
		opt(c)
	}
}
