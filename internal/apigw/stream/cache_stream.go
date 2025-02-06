package stream

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"encoding/json"
	"errors"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type cacheStream struct {
	service         *Service
	log             *logger.Log
	stream          jetstream.Stream
	js              jetstream.JetStream
	consumer        jetstream.Consumer
	consumerContext jetstream.ConsumeContext
}

func newCacheStream(ctx context.Context, service *Service) (*cacheStream, error) {
	s := &cacheStream{
		service: service,
		log:     service.log.New("cache"),
	}

	if err := s.createStream(ctx); err != nil {
		return nil, err
	}

	go func() {
		if err := s.Consume(ctx); err != nil {
			s.log.Error(err, "failed to consume")
		}
	}()

	s.log.Info("Started")

	return s, nil
}

func (s *cacheStream) jetstreamInit(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var err error
	s.js, err = jetstream.New(s.service.natsClient)
	if err != nil {
		s.log.Error(err, "Failed to connect to JetStream")
		return err
	}

	return nil
}

func (s *cacheStream) createStream(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.jetstreamInit(ctx); err != nil {
		return err
	}

	if s.js == nil {
		return errors.New("jetstream not initialized")
	}

	time.Sleep(3 * time.Second)

	var err error
	s.stream, err = s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "cache_stream",
		Subjects:  []string{"CACHE"},
		Retention: jetstream.WorkQueuePolicy,
		NoAck:     false,
	})
	if err != nil {
		s.log.Error(err, "Failed to create stream")
		return err
	}

	s.consumer, err = s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "cacher",
		Durable:       "cacher",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "CACHE",
	})
	if err != nil {
		s.log.Error(err, "Failed to create cache_stream consumer")
		return err
	}

	info, err := s.stream.Info(ctx)
	if err != nil {
		s.log.Error(err, "Failed to get stream info")
		return err
	}
	s.log.Debug("Stream info", "stream", info)

	return nil
}

func (s *cacheStream) Consume(ctx context.Context) error {
	var err error
	s.consumerContext, err = s.consumer.Consume(func(m jetstream.Msg) {
		m.InProgress()
		s.log.Debug("Received message", "subject", m.Subject(), "transaction_id", m.Headers().Get("Nats-Msg-Id"))
		document := &model.Document{}
		if err := json.Unmarshal(m.Data(), document); err != nil {
			s.log.Error(err, "Failed to unmarshal")
			m.Nak()
		}
		if err := s.service.kv.Doc.SaveSigned(ctx, &model.Document{
			TransactionID: document.TransactionID,
			Data:          document.Data,
			SealerBackend: document.SealerBackend,
		}); err != nil {
			s.log.Error(err, "Failed to cache signed document")
			m.Nak()
		}
		m.Ack()
	})
	if err != nil {
		s.log.Error(err, "Failed to consume")
		return err
	}

	return nil
}

func (s *cacheStream) close(ctx context.Context) error {
	s.consumerContext.Stop()
	s.log.Debug("Closing")

	return nil
}
