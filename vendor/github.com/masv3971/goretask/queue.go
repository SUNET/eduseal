package retask

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Queue queue
type Queue struct {
	redisClient *redis.Client
	queueName   string
}

// Enqueue adds a new task to the queue
func (q *Queue) Enqueue(ctx context.Context, data []byte) (*Job, error) {
	job := newJob(ctx, q.redisClient)

	task, err := makeWrapper(data, job.urn)
	if err != nil {
		return nil, err
	}
	if err := q.redisClient.LPush(ctx, q.queueName, task).Err(); err != nil {
		return nil, err
	}

	return job, nil
}

// Names returns a list of queues available
func (q *Queue) Names(ctx context.Context) ([]string, error) {
	return q.redisClient.Keys(ctx, "retaskqueue-*").Result()
}

// Length returns the length of the queue
func (q *Queue) Length(ctx context.Context) (int64, error) {
	return q.redisClient.LLen(ctx, q.queueName).Result()
}

// Wait waits for a task to be available in the queue
func (q *Queue) Wait(ctx context.Context) (*Task, error) {
	res, err := q.redisClient.BRPop(ctx, 0*time.Microsecond, q.queueName).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoResult
	}
	task, err := makeTask(ctx, res[1])
	if err != nil {
		return nil, err
	}

	return task, err
}

// Dequeue dequeues a task from the queue, if no task is available it returns ErrNoResult
func (q *Queue) Dequeue(ctx context.Context) (*Task, error) {
	res, err := q.redisClient.RPop(ctx, q.queueName).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoResult
	}
	task, err := makeTask(ctx, res)
	if err != nil {
		return nil, err
	}

	return task, err
}

// Send sends the result back to the producer.
func (q *Queue) Send(ctx context.Context, task Task, result []byte) error {
	res, err := makeWrapper(result, task.URN)
	if err != nil {
		return err
	}
	if err := q.redisClient.LPush(ctx, task.URN, res).Err(); err != nil {
		return err
	}
	if err := q.redisClient.Expire(ctx, task.URN, 60*time.Second).Err(); err != nil {
		return err
	}
	return nil
}
