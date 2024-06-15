package simplequeue

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"encoding/json"

	retask "github.com/masv3971/goretask"
	"go.opentelemetry.io/otel/codes"
)

// EduSealAddSigned is the ladok unsigned queue
type EduSealAddSigned struct {
	service *Service
	log     *logger.Log
	*retask.Queue
}

// NewEduSealAddSigned creates a new EduSeal unsigned queue
func NewEduSealAddSigned(ctx context.Context, service *Service, queueName string, log *logger.Log) (*EduSealAddSigned, error) {
	eduSealAddSigned := &EduSealAddSigned{
		service: service,
		log:     log,
	}

	eduSealAddSigned.Queue = eduSealAddSigned.service.queueClient.NewQueue(ctx, queueName)

	eduSealAddSigned.log.Info("Started")

	return eduSealAddSigned, nil
}

// Enqueue publishes a document to the queue
func (s *EduSealAddSigned) Enqueue(ctx context.Context, message any) (*retask.Job, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealAddSigned:Enqueue")
	defer span.End()

	s.log.Debug("Enqueue add signed pdf")

	data, err := json.Marshal(message)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return s.Queue.Enqueue(ctx, data)
}

// Dequeue dequeues a document from the queue
func (s *EduSealAddSigned) Dequeue(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealAddSigned:Dequeue")
	defer span.End()
	return nil
}

// Wait waits for the next message
func (s *EduSealAddSigned) Wait(ctx context.Context) (*retask.Task, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealAddSigned:Wait")
	defer span.End()

	task, err := s.Queue.Wait(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return task, nil
}

// Worker is the worker
func (s *EduSealAddSigned) Worker(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSealAddSigned:Worker")
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
			return err
		case task := <-taskChan:
			s.log.Info("Got task", "task", task.Data)
			document := &model.Document{}
			if err := json.Unmarshal([]byte(task.Data), document); err != nil {
				span.SetStatus(codes.Error, err.Error())
				s.log.Error(err, "Unmarshal failed")
			}
			if err := s.service.kv.Doc.SaveSigned(ctx, document); err != nil {
				span.SetStatus(codes.Error, err.Error())
				s.log.Error(err, "SaveSigned failed")
			}

		case <-ctx.Done():
			s.log.Info("Stopped worker")
			return nil
		}
	}
}
