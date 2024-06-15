package simplequeue

import (
	"context"
	"eduseal/pkg/logger"
	"encoding/json"

	retask "github.com/masv3971/goretask"
	"go.opentelemetry.io/otel/codes"
)

// EduSealValidate is the EduSeal unsigned queue
type EduSealValidate struct {
	service   *Service
	queueName string
	log       *logger.Log
	*retask.Queue
}

// NewEduSealValidate creates a new EduSeal unsigned queue
func NewEduSealValidate(ctx context.Context, service *Service, queueName string, log *logger.Log) (*EduSealValidate, error) {
	eduSealValidate := &EduSealValidate{
		service: service,
		log:     log,
	}

	eduSealValidate.Queue = eduSealValidate.service.queueClient.NewQueue(ctx, queueName)

	eduSealValidate.log.Info("Started")

	return eduSealValidate, nil
}

// Enqueue publishes a document to the queue
func (s *EduSealValidate) Enqueue(ctx context.Context, message any) (*retask.Job, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealValidate:Enqueue")
	defer span.End()

	data, err := json.Marshal(message)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return s.Queue.Enqueue(ctx, data)
}

// Wait waits for the next message
func (s *EduSealValidate) Wait(ctx context.Context) (*retask.Task, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealValidate:Wait")
	defer span.End()

	return s.Queue.Wait(ctx)
}

// Dequeue dequeues a document from the queue
func (s *EduSealValidate) Dequeue(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealValidate:Dequeue")
	defer span.End()
	return nil
}
