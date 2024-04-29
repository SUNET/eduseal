package simplequeue

import (
	"context"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"

	retask "github.com/masv3971/goretask"
	"github.com/redis/go-redis/v9"
)

type queue interface {
	Enqueue(ctx context.Context, message any) (*retask.Job, error)
	Dequeue(ctx context.Context) error
	Wait(ctx context.Context) (*retask.Task, error)
	Worker(ctx context.Context) error
}

// Service is the service object for queue
type Service struct {
	queueClient *retask.Client
	redisClient *redis.Client
	kv          *kvclient.Client
	tp          *trace.Tracer
	log         *logger.Log
	cfg         *model.Cfg

	LadokAddSigned queue
	LadokDelSigned queue
}

// New creates a new queue service
func New(ctx context.Context, kv *kvclient.Client, tracer *trace.Tracer, cfg *model.Cfg, log *logger.Log) (*Service, error) {
	service := &Service{
		redisClient: kv.RedisClient,
		kv:          kv,
		tp:          tracer,
		log:         log,
		cfg:         cfg,
	}

	var err error
	service.queueClient, err = retask.New(ctx, service.redisClient)
	if err != nil {
		return nil, err
	}

	service.LadokAddSigned, err = NewLadokAddSigned(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealAddSigned.Name, service.log.New("LadokAddSigned"))
	if err != nil {
		return nil, err
	}
	service.LadokDelSigned, err = NewLadokDelSigned(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealDelSigned.Name, service.log.New("LadokDelSigned"))
	if err != nil {
		return nil, err
	}

	go service.LadokAddSigned.Worker(ctx)
	go service.LadokDelSigned.Worker(ctx)

	return service, nil
}

// Close closes the service
func (s *Service) Close(ctx context.Context) error {
	s.log.Info("Stopped")
	ctx.Done()
	return nil
}
