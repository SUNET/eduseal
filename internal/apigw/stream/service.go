package stream

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service is the stream service object
type Service struct {
	log        *logger.Log
	cfg        *model.Cfg
	natsClient *nats.Conn
	kv         *kvclient.Client
	probeStore *v1_status.StatusProbeStore
	statusTick *time.Ticker
	tp         *trace.Tracer

	Seal  *sealStream
	Cache *cacheStream
}

// New creates a new stream service
func New(ctx context.Context, kv *kvclient.Client, tp *trace.Tracer, cfg *model.Cfg, log *logger.Log) (*Service, error) {
	s := &Service{
		log:        log,
		cfg:        cfg,
		kv:         kv,
		probeStore: &v1_status.StatusProbeStore{},
		statusTick: time.NewTicker(time.Second * 10),
		tp:         tp,
	}

	if err := s.connect(ctx); err != nil {
		return nil, err
	}

	s.probe(ctx)

	var err error

	s.Cache, err = newCacheStream(ctx, s)
	if err != nil {
		s.log.Error(err, "Failed to create cache stream")
		return nil, err
	}

	s.Seal, err = newSealStream(ctx, s)
	if err != nil {
		s.log.Error(err, "Failed to create sign stream")
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.statusTick.C:
				s.log.Info("Checking status")
				s.probe(ctx)
			}
		}
	}()

	s.log.Info("Started")

	return s, nil
}

func (s *Service) connect(ctx context.Context) error {
	_, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	servers := strings.Join(s.cfg.Common.Queue.Addr, ",")

	s.log.Info("Connecting to NATS", "servers", servers)

	var err error
	s.natsClient, err = nats.Connect(
		servers,
		nats.Timeout(2*time.Second),
		nats.MaxReconnects(10),
		nats.RetryOnFailedConnect(true),
		nats.ReconnectWait(2*time.Second),
		nats.Name("apigw"),
		nats.UserInfo(s.cfg.Common.Queue.Username, s.cfg.Common.Queue.Password),
	)
	if err != nil {
		s.log.Error(err, "Failed to connect to NATS")
		return err
	}

	return nil
}

func (s *Service) probe(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	s.probeStore.PreviousResult = &v1_status.StatusProbe{
		Name:          "stream/nats",
		Healthy:       false,
		Message:       "Not connected",
		LastCheckedTS: timestamppb.Now(),
	}
	if s.natsClient.IsConnected() {
		s.probeStore.PreviousResult.Message = "Connected"
		s.probeStore.PreviousResult.Healthy = true
	}
}

// Status returns the status of the database
func (s *Service) Status(ctx context.Context) *v1_status.StatusProbe {
	ctx, span := s.tp.Start(ctx, "stream:Status")
	defer span.End()

	return s.probeStore.PreviousResult
}

// Close closes the stream service
func (s *Service) Close(ctx context.Context) error {
	s.natsClient.Close()
	s.log.Info("Closed")
	ctx.Done()
	return nil
}
