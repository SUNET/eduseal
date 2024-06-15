package retask

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Job type containing the result from the workers.
type Job struct {
	urn         string
	redisClient *redis.Client
	Result      *Task
}

var testUUID = ""

func newJob(ctx context.Context, redisClient *redis.Client) *Job {
	var urn string

	if testUUID == "" {
		urn = uuid.NewString()
	} else {
		urn = testUUID
	}

	return &Job{
		urn:         urn,
		redisClient: redisClient,
	}
}

// Result returns the result of the job
//func (job *Job) Result(ctx context.Context) (*Task, error) {
//	if job.result != nil {
//		return job.result, nil
//	}
//	data := job.redisClient.RPop(context.Background(), job.urn).Val()
//	if data != "" {
//		job.redisClient.Del(context.Background(), job.urn)
//		var err error
//		job.result, err = makeTask(ctx, data)
//		if err != nil {
//			return nil, err
//		}
//		return job.result, nil
//	}
//	return nil, nil
//}

// Wait waits for a task to be available in the queue
func (job *Job) Wait(ctx context.Context) (bool, error) {
	if job.Result != nil {
		fmt.Println("result", job.Result)
		return true, nil
	}

	res, err := job.redisClient.BRPop(ctx, 0*time.Microsecond, job.urn).Result()
	if err != nil {
		return false, err
	}
	if len(res) == 0 {
		return false, ErrNoResult
	}
	fmt.Println("res", res)
	task, err := makeTaskFromWait(ctx, res)
	if err != nil {
		return false, err
	}
	fmt.Println("task", task)
	if task != nil {
		if err := job.redisClient.Del(ctx, job.urn).Err(); err != nil {
			return false, err
		}
		job.Result = task
		return true, nil
	}

	return false, err
}
