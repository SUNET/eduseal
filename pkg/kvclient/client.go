package kvclient

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/certreloader"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client holds the kv object
type Client struct {
	backend      Backend
	cfg          *model.Cfg
	log          *logger.Log
	probeStore   *v1_status.StatusProbeStore
	tp           *trace.Tracer
	statusTick   *time.Ticker
	certReloader *certreloader.CertReloader

	Doc *Doc
}

// New creates a new instance of kv
func New(ctx context.Context, cfg *model.Cfg, tracer *trace.Tracer, log *logger.Log) (*Client, error) {
	c := &Client{
		cfg:        cfg,
		log:        log,
		probeStore: &v1_status.StatusProbeStore{},
		tp:         tracer,
	}

	kvCfg := &cfg.Common.KV

	var err error

	if kvCfg.TLS.Enabled && kvCfg.TLS.CertFilePath != "" {
		c.certReloader, err = certreloader.New(kvCfg.TLS.CertFilePath, kvCfg.TLS.KeyFilePath, log)
		if err != nil {
			return nil, fmt.Errorf("kv cert reloader: %w", err)
		}
	}

	kvType := kvCfg.Type
	if kvType == "" {
		kvType = "valkey"
	}

	switch kvType {
	case "valkey":
		c.backend, err = newValkeyBackend(kvCfg, c.certReloader)
	case "redict":
		c.backend, err = newRedictBackend(kvCfg, c.certReloader)
	default:
		c.closeCertReloader()
		return nil, fmt.Errorf("unsupported kv type: %q (must be \"valkey\" or \"redict\")", kvType)
	}
	if err != nil {
		c.closeCertReloader()
		return nil, err
	}

	c.statusTick = time.NewTicker(time.Second * 10)

	c.probe(ctx)

	c.Doc = &Doc{client: c, key: "doc:%s:%s"}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.statusTick.C:
				c.log.Info("Checking status")
				c.probe(ctx)
			}
		}
	}()

	c.log.Info("Started")

	return c, nil
}

func (c *Client) probe(ctx context.Context) {
	c.probeStore.PreviousResult = &v1_status.StatusProbe{
		Name:          "kv",
		Healthy:       true,
		Message:       "OK",
		LastCheckedTS: timestamppb.Now(),
	}
	if err := c.backend.Ping(ctx); err != nil {
		c.probeStore.PreviousResult.Message = err.Error()
		c.probeStore.PreviousResult.Healthy = false
	}
}

// Status returns the status of the database
func (c *Client) Status(ctx context.Context) *v1_status.StatusProbe {
	_, span := c.tp.Start(ctx, "kv:Status")
	defer span.End()

	return c.probeStore.PreviousResult
}

func (c *Client) closeCertReloader() {
	if c.certReloader != nil {
		c.certReloader.Close()
		c.certReloader = nil
	}
}

// Close closes the connection to the database
func (c *Client) Close(ctx context.Context) error {
	c.statusTick.Stop()
	c.closeCertReloader()
	c.backend.Close()
	return nil
}
