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
}

// Service is the service object for queue
type Service struct {
	redisClient *redis.Client
	queueClient *retask.Client
	tp          *trace.Tracer
	log         *logger.Log
	cfg         *model.Cfg

	EduSealSign           queue
	EduSealValidate       queue
	EduSealDelSigned      queue
	EduSealPersistentSave queue
}

// New creates a new queue service
func New(ctx context.Context, kv *kvclient.Client, tracer *trace.Tracer, cfg *model.Cfg, log *logger.Log) (*Service, error) {
	service := &Service{
		redisClient: kv.RedisClient,
		tp:          tracer,
		log:         log,
		cfg:         cfg,
	}
	var err error
	service.queueClient, err = retask.New(ctx, service.redisClient)
	if err != nil {
		return nil, err
	}

	service.EduSealSign, err = NewEduSealSign(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealSeal.Name, service.log.New("EduSealSign"))
	if err != nil {
		return nil, err
	}

	service.EduSealValidate, err = NewEduSealValidate(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealValidate.Name, service.log.New("EduSealValidate"))
	if err != nil {
		return nil, err
	}

	service.EduSealDelSigned, err = NewEduSealDelSigned(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealDelSealed.Name, service.log.New("EduSealDelSigned"))
	if err != nil {
		return nil, err
	}

	service.EduSealPersistentSave, err = NewEduSealPersistentSave(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealPersistentSave.Name, service.log.New("EduSealPersistentSave"))
	if err != nil {
		return nil, err
	}

	service.log.Info("Started")

	return service, nil
}

// Close closes the service
func (s *Service) Close(ctx context.Context) error {
	s.log.Info("Stopped")
	return nil
}
