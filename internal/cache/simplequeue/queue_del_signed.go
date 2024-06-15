package simplequeue

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"encoding/json"

	retask "github.com/masv3971/goretask"
	"go.opentelemetry.io/otel/codes"
)

// EduSealDelSigned holds the ladok delete signed queue
type EduSealDelSigned struct {
	service *Service
	log     *logger.Log
	*retask.Queue
}

// NewEduSealDelSigned creates a new ladok delete signed queue
func NewEduSealDelSigned(ctx context.Context, service *Service, queueName string, log *logger.Log) (*EduSealDelSigned, error) {
	eduSealDelSigned := &EduSealDelSigned{
		service: service,
		log:     log,
	}

	eduSealDelSigned.Queue = eduSealDelSigned.service.queueClient.NewQueue(ctx, queueName)

	eduSealDelSigned.log.Info("Started")

	return eduSealDelSigned, nil
}

// Enqueue publishes a document to the queue
func (s *EduSealDelSigned) Enqueue(ctx context.Context, message any) (*retask.Job, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealDelSigned:Enqueue")
	defer span.End()

	data, err := json.Marshal(message)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return s.Queue.Enqueue(ctx, data)
}

// Dequeue dequeues a document from the queue
func (s *EduSealDelSigned) Dequeue(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealDelSigned:Dequeue")
	defer span.End()
	return nil
}

// Wait waits for the next message
func (s *EduSealDelSigned) Wait(ctx context.Context) (*retask.Task, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealDelSigned:Wait")
	defer span.End()

	task, err := s.Queue.Wait(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return task, nil
}

// Worker is the worker
func (s *EduSealDelSigned) Worker(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealDelSigned:Worker")
	defer span.End()

	var (
		taskChan = make(chan *retask.Task)
		errChan  = make(chan error)
	)

	go func() {
		for {
			task, err := s.Wait(ctx)
			if err != nil {
				errChan <- err
			}
			taskChan <- task
		}
	}()

	for {
		select {
		case err := <-errChan:
			s.log.Error(err, "Worker failed")
		case task := <-taskChan:
			s.log.Info("Got task", "task", task)
			document := &model.Document{}
			if err := json.Unmarshal([]byte(task.Data), document); err != nil {
				span.SetStatus(codes.Error, err.Error())
				s.log.Error(err, "Unmarshal failed")
			}

			if err := s.service.kv.Doc.DelSigned(ctx, document.TransactionID); err != nil {
				span.SetStatus(codes.Error, err.Error())
				s.log.Error(err, "DelSigned failed")
			}
		case <-ctx.Done():
			s.log.Info("Stopped worker")
			return nil
		}
	}
}
