package simplequeue

import (
	"context"
	"eduseal/pkg/logger"
	"encoding/json"

	retask "github.com/masv3971/goretask"
	"github.com/masv3971/gosunetca/types"
	"go.opentelemetry.io/otel/codes"
)

// LadokAddSigned is the ladok unsigned queue
type LadokAddSigned struct {
	service *Service
	log     *logger.Log
	*retask.Queue
}

// NewLadokAddSigned creates a new ladok unsigned queue
func NewLadokAddSigned(ctx context.Context, service *Service, queueName string, log *logger.Log) (*LadokAddSigned, error) {
	ladokAddSigned := &LadokAddSigned{
		service: service,
		log:     log,
	}

	ladokAddSigned.Queue = ladokAddSigned.service.queueClient.NewQueue(ctx, queueName)

	ladokAddSigned.log.Info("Started")

	return ladokAddSigned, nil
}

// Enqueue publishes a document to the queue
func (s *LadokAddSigned) Enqueue(ctx context.Context, message any) (*retask.Job, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:LadokAddSigned:Enqueue")
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
func (s *LadokAddSigned) Dequeue(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:LadokAddSigned:Dequeue")
	defer span.End()
	return nil
}

// Wait waits for the next message
func (s *LadokAddSigned) Wait(ctx context.Context) (*retask.Task, error) {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:LadokAddSigned:Wait")
	defer span.End()

	task, err := s.Queue.Wait(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return task, nil
}

// Worker is the worker
func (s *LadokAddSigned) Worker(ctx context.Context) error {
	ctx, span := s.service.tp.Start(ctx, "simplequeue:LadokAddSigned:Worker")
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
			document := &types.Document{}
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
