package kvclient

import "context"

// Backend abstracts the low-level key/value store operations so that
// different engines (Valkey, Redict/Redis, …) can be used interchangeably.
type Backend interface {
	// HSet sets one or more hash fields.
	HSet(ctx context.Context, key string, fields map[string]string) error
	// HGetAll returns all fields and values of a hash.
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	// Expire sets a TTL (in seconds) on the given key.
	Expire(ctx context.Context, key string, seconds int64) error
	// Exists returns true when the key exists.
	Exists(ctx context.Context, key string) (bool, error)
	// HDel removes one or more fields from a hash.
	HDel(ctx context.Context, key string, fields ...string) error
	// Ping checks connectivity.
	Ping(ctx context.Context) error
	// Close releases any resources held by the backend.
	Close()
}
