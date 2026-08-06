package stream

import (
	"context"
	"eduseal/pkg/logger"
	"time"

	"go.opentelemetry.io/otel/codes"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type sealStream struct {
	service  *Service
	log      *logger.Log
	stream   jetstream.Stream
	js       jetstream.JetStream
	consumer jetstream.Consumer
}

func newSealStream(ctx context.Context, service *Service) (*sealStream, error) {
	s := &sealStream{
		service: service,
		log:     service.log.New("seal"),
	}

	if err := s.createStream(ctx); err != nil {
		return nil, err
	}

	s.log.Info("Started")

	return s, nil
}

// Publish publishes a message to the stream with retry logic
func (s *sealStream) Publish(ctx context.Context, payload []byte, transactionID string) error {
	ctx, span := s.service.tp.Start(ctx, "stream:seal:PDFSign")
	defer span.End()

	s.log.Info("Publishing", "transaction_id", transactionID)

	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		pubCtx, cancel := context.WithTimeout(ctx, 2*time.Second)

		ack, err := s.js.PublishMsg(pubCtx, &nats.Msg{
			Subject: "SEAL",
			Header: map[string][]string{
				"Nats-Msg-Id": {transactionID},
			},
			Data: payload,
			Sub: &nats.Subscription{
				Queue: "sealers",
			},
		})
		cancel()

		if err == nil {
			s.log.Debug("Published", "transaction_id", transactionID, "ack", ack, "attempt", attempt)
			return nil
		}

		lastErr = err
		s.log.Error(err, "Failed to publish, retrying", "transaction_id", transactionID, "attempt", attempt, "max_retries", maxRetries)

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				span.SetStatus(codes.Error, ctx.Err().Error())
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
	}

	span.SetStatus(codes.Error, lastErr.Error())
	s.log.Error(lastErr, "Failed to publish after all retries", "transaction_id", transactionID)
	return lastErr
}

func (s *sealStream) createStream(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var err error
	s.js, err = jetstream.New(s.service.natsClient)
	if err != nil {
		s.log.Error(err, "Failed to connect to JetStream")
		return err
	}

	s.stream, err = s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "seal_stream",
		Subjects:  []string{"SEAL"},
		Retention: jetstream.WorkQueuePolicy,
		NoAck:     false,
		Replicas:  s.service.cfg.Common.Queue.StreamReplicas,
	})
	if err != nil {
		s.log.Error(err, "Failed to create stream")
		return err
	}

	consumers := s.stream.ListConsumers(ctx)
	s.log.Debug("Consumers", "consumers", consumers)

	s.consumer, err = s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "sealer",
		Durable:       "sealer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "SEAL",
		MaxDeliver:    5,
		BackOff:       []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second, 3 * time.Second},
	})
	if err != nil {
		s.log.Error(err, "Failed to create seal_stream consumer")
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

func (s *sealStream) close(ctx context.Context) error {
	s.log.Debug("Closing")

	return nil
}
