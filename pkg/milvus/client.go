package milvus

import (
	"fmt"
	"strings"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Client creates a new Milvus client (not bound to any collection)
func (m *Milvus) Client(address string, token ...string) (*Client, error) {
	return m.createClient(address, "", token...)
}

// ClientWithCollection creates a new Milvus client bound to a specific collection
// This follows Locust's pattern where client is tied to a collection
func (m *Milvus) ClientWithCollection(address, collectionName string, token ...string) (*Client, error) {
	return m.createClient(address, collectionName, token...)
}

func (m *Milvus) createClient(address, collectionName string, token ...string) (*Client, error) {
	ctx := m.vu.Context()

	// Create client config
	clientConfig := DefaultClientConfig()
	clientConfig.Address = address
	clientConfig.DefaultCollection = collectionName

	// Parse token if provided (format: "username:password")
	if len(token) > 0 && token[0] != "" {
		parts := strings.Split(token[0], ":")
		if len(parts) == 2 {
			clientConfig.Username = parts[0]
			clientConfig.Password = parts[1]
		}
	}

	milvusConfig := &milvusclient.ClientConfig{
		Address: clientConfig.Address,
	}

	if clientConfig.Username != "" {
		milvusConfig.Username = clientConfig.Username
		milvusConfig.Password = clientConfig.Password
	}

	c, err := milvusclient.New(ctx, milvusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %v", err)
	}

	return &Client{
		client:            c,
		ctx:               ctx,
		vu:                m.vu,
		config:            clientConfig,
		defaultCollection: collectionName,
	}, nil
}

// Close closes the Milvus client connection
func (c *Client) Close() error {
	return c.client.Close(c.ctx)
}

// GetClient returns a cached client for the given address and collection.
// If no cached client exists, it creates a new one and caches it.
// This method is designed for VU-level client reuse to avoid repeated
// DescribeCollection calls (Milvus SDK caches schema per client).
//
// Usage in k6:
//
//	import milvus from 'k6/x/milvus';
//	export default function() {
//	    const client = milvus.getClient(host, collection, token);
//	    client.upsert({...});
//	    // Don't call client.close() - let it be reused
//	}
func (m *Milvus) GetClient(address, collectionName string, token ...string) (*Client, error) {
	// Build cache key
	key := address + ":" + collectionName

	// Check if we have a cached client
	if client, ok := m.clients[key]; ok {
		// Update context for current iteration
		client.ctx = m.vu.Context()
		return client, nil
	}

	// Create new client
	client, err := m.createClient(address, collectionName, token...)
	if err != nil {
		return nil, err
	}

	// Cache it
	m.clients[key] = client
	return client, nil
}
