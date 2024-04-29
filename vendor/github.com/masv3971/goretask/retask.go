package retask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func newURN() string {
	return uuid.NewString()
}

// Task is a serialized representation of a task in the queue
type Task struct {
	Data string `json:"_data"`
	URN  string `json:"urn"`
	raw  string
}

// Client is a client for interacting with the queue
type Client struct {
	redisClient *redis.Client
	queueName   string
	uuidFunc    func() string
	currentURN  string
}

func (c *Client) setNewURN() {
	c.currentURN = c.uuidFunc()
}

// New creates a new client for interacting with the queue
func New(ctx context.Context, redisClient *redis.Client) (*Client, error) {
	client := &Client{
		redisClient: redisClient,
		uuidFunc:    newURN,
	}

	return client, nil
}

// NewQueue creates a new queue
func (c *Client) NewQueue(ctx context.Context, queueName string) *Queue {
	q := &Queue{
		redisClient: c.redisClient,
		queueName:   fmt.Sprintf("retaskqueue-%s", queueName),
	}
	return q
}

func makeTask(ctx context.Context, data string) (*Task, error) {
	task := &Task{
		raw: data,
	}
	if err := json.Unmarshal([]byte(data), task); err != nil {
		return nil, err
	}
	return task, nil
}

func makeTaskFromWait(ctx context.Context, data []string) (*Task, error) {
	if len(data) == 0 {
		return nil, ErrNoResult
	}
	task := &Task{
		raw:  data[1],
		URN:  data[0],
		Data: data[1],
	}
	return task, nil
}

func makeWrapper(data []byte, urn string) (string, error) {
	wrapper := map[string]any{
		"_data": string(data),
		"urn":   urn,
	}
	task, err := json.Marshal(wrapper)
	if err != nil {
		return "", err
	}
	return string(task), nil
}
