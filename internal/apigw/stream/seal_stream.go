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
	service         *Service
	log             *logger.Log
	stream          jetstream.Stream
	js              jetstream.JetStream
	consumer        jetstream.Consumer
	consumerContext jetstream.ConsumeContext
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

// Publish publishes a message to the stream
func (s *sealStream) Publish(ctx context.Context, payload []byte, transactionID string) error {
	ctx, span := s.service.tp.Start(ctx, "stream:seal:PDFSign")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	s.log.Info("Publishing", "transaction_id", transactionID)

	ack, err := s.js.PublishMsg(ctx, &nats.Msg{
		Subject: "SEAL",
		Header: map[string][]string{
			"Nats-Msg-Id": {transactionID},
		},
		Data: payload,
		Sub: &nats.Subscription{
			Queue: "sealers",
		},
	})

	//ack, err := s.js.Publish(ctx, "SEAL", payload, jetstream.WithMsgID(transactionID))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		s.log.Error(err, "Failed to publish")
		return err
	}

	s.log.Debug("Published", "transaction_id", transactionID, "ack", ack)

	//	select {
	//	case <-s.service.stream.PublishAsyncComplete():
	//		s.log.Debug("Published", "transaction_id", transactionID)
	//	case <-time.After(5 * time.Second):
	//		s.log.Debug("Failed to publish", "transaction_id", transactionID)
	//	}

	return nil
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
