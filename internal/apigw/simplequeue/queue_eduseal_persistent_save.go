package simplequeue

import (
	"context"
	"eduseal/pkg/logger"
	"encoding/json"

	retask "github.com/masv3971/goretask"
	"go.opentelemetry.io/otel/codes"
)

// EduSealPersistentSave holds the EduSeal delete signed queue
type EduSealPersistentSave struct {
	service *Service
	log     *logger.Log
	*retask.Queue
}

// NewEduSealPersistentSave creates a new EduSeal delete signed queue
func NewEduSealPersistentSave(ctx context.Context, service *Service, queueName string, log *logger.Log) (*EduSealDelSigned, error) {
	eduSealPersistentSave := &EduSealDelSigned{
		service: service,
		log:     log,
	}

	eduSealPersistentSave.Queue = eduSealPersistentSave.service.queueClient.NewQueue(ctx, queueName)

	eduSealPersistentSave.log.Info("Started")

	return eduSealPersistentSave, nil
}

// Enqueue publishes a document to the queue
func (s *EduSealPersistentSave) Enqueue(ctx context.Context, message any) (*retask.Job, error) {
	s.log.Info("Enqueue delete signed pdf")
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealPersistentSave:Enqueue")
	defer span.End()

	data, err := json.Marshal(message)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return s.Queue.Enqueue(ctx, data)
}

// Dequeue dequeues a document from the queue
func (s *EduSealPersistentSave) Dequeue(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealPersistentSave:Dequeue")
	defer span.End()
	return nil
}

// Wait waits for the next message
func (s *EduSealPersistentSave) Wait(ctx context.Context) (*retask.Task, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealPersistentSave:Wait")
	defer span.End()

	task, err := s.Queue.Wait(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return task, nil
}
