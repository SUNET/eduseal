package kvclient

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// MetricSigning holds the signing metric object
type MetricSigning struct {
	client *Client
	key    string
}

// Inc increments the signing metric
func (m *MetricSigning) Inc(ctx context.Context) error {
	return m.client.RedictCC.Incr(ctx, m.key).Err()
}

// Get returns the signing metric
func (m *MetricSigning) Get(ctx context.Context) (int64, error) {
	r, err := m.client.RedictCC.Get(ctx, m.key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return r, err
}

// MetricFetching holds the fetching metric object
type MetricFetching struct {
	client *Client
	key    string
}

// Inc increments the signing metric
func (m *MetricFetching) Inc(ctx context.Context) error {
	return m.client.RedictCC.Incr(ctx, m.key).Err()
}

// Get returns the fetching metric
func (m *MetricFetching) Get(ctx context.Context) (int64, error) {
	r, err := m.client.RedictCC.Get(ctx, m.key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return r, err
}

// MetricValidations holds the validations metric object
type MetricValidations struct {
	client *Client
	key    string
}

// Inc increments the validations metric
func (m *MetricValidations) Inc(ctx context.Context) error {
	return m.client.RedictCC.Incr(ctx, m.key).Err()
}

// Get returns the validations metric
func (m *MetricValidations) Get(ctx context.Context) (int64, error) {
	r, err := m.client.RedictCC.Get(ctx, m.key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return r, err
}
