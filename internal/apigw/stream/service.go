package stream

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"fmt"
	"os"
	"path/filepath"
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

	opts := []nats.Option{
		nats.Timeout(2 * time.Second),
		nats.MaxReconnects(100),
		nats.RetryOnFailedConnect(true),
		nats.ReconnectWait(2 * time.Second),
		nats.Name("apigw"),
		nats.UserInfo(s.cfg.Common.Queue.Username, s.cfg.Common.Queue.Password),
	}

	if s.cfg.Common.Queue.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: s.cfg.Common.Queue.TLS.ServerName,
		}

		if s.cfg.Common.Queue.TLS.RootCAPath != "" {
			caCertByte, err := os.ReadFile(filepath.Clean(s.cfg.Common.Queue.TLS.RootCAPath))
			if err != nil {
				return fmt.Errorf("failed to read queue root CA: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCertByte) {
				return fmt.Errorf("failed to parse queue root CA at %q", s.cfg.Common.Queue.TLS.RootCAPath)
			}
			tlsConfig.RootCAs = caCertPool
		}

		clientCert, err := tls.LoadX509KeyPair(
			filepath.Clean(s.cfg.Common.Queue.TLS.CertFilePath),
			filepath.Clean(s.cfg.Common.Queue.TLS.KeyFilePath),
		)
		if err != nil {
			return fmt.Errorf("failed to load queue client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}

		opts = append(opts, nats.Secure(tlsConfig))
	}

	var err error
	s.natsClient, err = nats.Connect(servers, opts...)
	if err != nil {
		s.log.Error(err, "Failed to connect to NATS")
		return err
	}

	return nil
}

func (s *Service) probe(ctx context.Context) {
	_, cancel := context.WithTimeout(ctx, 1*time.Second)
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
	_, span := s.tp.Start(ctx, "stream:Status")
	defer span.End()

	return s.probeStore.PreviousResult
}

// Close closes the stream service
func (s *Service) Close(ctx context.Context) error {
	if err := s.Cache.close(ctx); err != nil {
		return err
	}

	if err := s.Seal.close(ctx); err != nil {
		return err
	}

	s.natsClient.Close()
	s.log.Info("Closed")
	ctx.Done()
	return nil
}
