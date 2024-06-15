package simplequeue

import (
	"context"
	"eduseal/internal/persistent/db"
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
	db          *db.Service
	tp          *trace.Tracer
	log         *logger.Log
	cfg         *model.Cfg

	EduSealPersistentSave queue
}

// New creates a new queue service
func New(ctx context.Context, kv *kvclient.Client, db *db.Service, tracer *trace.Tracer, cfg *model.Cfg, log *logger.Log) (*Service, error) {
	service := &Service{
		redisClient: kv.RedisClient,
		kv:          kv,
		db:          db,
		tp:          tracer,
		log:         log,
		cfg:         cfg,
	}

	var err error
	service.queueClient, err = retask.New(ctx, service.redisClient)
	if err != nil {
		return nil, err
	}

	service.EduSealPersistentSave, err = NewEduSealPersistentSave(ctx, service, cfg.Common.Queues.SimpleQueue.EduSealPersistentSave.Name, service.log.New("EduSealPersistentSave"))
	if err != nil {
		return nil, err
	}

	go service.EduSealPersistentSave.Worker(ctx)

	return service, nil
}

// Close closes the service
func (s *Service) Close(ctx context.Context) error {
	s.log.Info("Stopped")
	ctx.Done()
	return nil
}
