package kvclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/redis/go-redis/v9"
	//"codeberg.org/redict/go-redic"
)

// Client holds the kv object
type Client struct {
	RedictCC   *redis.ClusterClient
	cfg        *model.Cfg
	log        *logger.Log
	probeStore *v1_status.StatusProbeStore
	tp         *trace.Tracer
	statusTick *time.Ticker

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

	//clientCert, err := tls.LoadX509KeyPair(cfg.APIGW.ClientCert.CertFilePath, cfg.APIGW.ClientCert.KeyFilePath)
	//if err != nil {
	//	return nil, err
	//}

	clusterOpts := &redis.ClusterOptions{
		Addrs:    cfg.Common.Redict.Nodes,
		Password: cfg.Common.Redict.Password,
	}

	if cfg.Common.Redict.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Common.Redict.TLS.ServerName,
		}

		if cfg.Common.Redict.TLS.RootCAPath != "" {
			caCertByte, err := os.ReadFile(filepath.Clean(cfg.Common.Redict.TLS.RootCAPath))
			if err != nil {
				return nil, fmt.Errorf("failed to read redict root CA: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCertByte) {
				return nil, fmt.Errorf("failed to parse redict root CA at %q", cfg.Common.Redict.TLS.RootCAPath)
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Config validation guarantees both are set when TLS is enabled.
		clientCert, err := tls.LoadX509KeyPair(
			filepath.Clean(cfg.Common.Redict.TLS.CertFilePath),
			filepath.Clean(cfg.Common.Redict.TLS.KeyFilePath),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load redict client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}

		clusterOpts.TLSConfig = tlsConfig
	}

	c.RedictCC = redis.NewClusterClient(clusterOpts)

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
	_, err := c.RedictCC.Ping(ctx).Result()
	if err != nil {
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

// Close closes the connection to the database
func (c *Client) Close(ctx context.Context) error {
	return c.RedictCC.Close()
}
