package simplequeue

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"encoding/json"

	retask "github.com/masv3971/goretask"
	"go.opentelemetry.io/otel/codes"
)

// EduSealPersistentSave holds the ladok delete signed queue
type EduSealPersistentSave struct {
	service *Service
	log     *logger.Log
	*retask.Queue
}

// NewEduSealPersistentSave creates a new ladok delete signed queue
func NewEduSealPersistentSave(ctx context.Context, service *Service, queueName string, log *logger.Log) (*EduSealPersistentSave, error) {
	eduSealPersistentSave := &EduSealPersistentSave{
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
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSeal:PersistentSave:Enqueue")
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
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSeal:PersistentSave:Dequeue")
	defer span.End()
	return nil
}

// Wait waits for the next message
func (s *EduSealPersistentSave) Wait(ctx context.Context) (*retask.Task, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSeal:PersistentSave:Wait")
	defer span.End()

	task, err := s.Queue.Wait(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return task, nil
}

// Worker is the worker
func (s *EduSealPersistentSave) Worker(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:EduSeal:PersistentSave:Worker")
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
			if s.service.cfg.Common.Mongo.Disable {
				s.log.Info("Not saving document")
			} else {
				if err := s.service.db.EduSealDocumentColl.Save(ctx, document); err != nil {
					span.SetStatus(codes.Error, err.Error())
					s.log.Error(err, "SaveSigned failed")
				}
			}

		case <-ctx.Done():
			s.log.Info("Stopped worker")
			return nil
		}
	}
}
