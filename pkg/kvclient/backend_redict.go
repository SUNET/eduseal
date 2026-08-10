package kvclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"eduseal/pkg/certreloader"
	"eduseal/pkg/model"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
)

type redictBackend struct {
	client redis.UniversalClient
}

func newRedictBackend(cfg *model.KV, cr *certreloader.CertReloader) (*redictBackend, error) {
	opts := &redis.UniversalOptions{
		Addrs:    cfg.Nodes,
		Password: cfg.Password,
	}

	if cfg.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if cfg.TLS.RootCAPath != "" {
			caCertByte, err := os.ReadFile(filepath.Clean(cfg.TLS.RootCAPath))
			if err != nil {
				return nil, fmt.Errorf("failed to read kv root CA: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCertByte) {
				return nil, fmt.Errorf("failed to parse kv root CA at %q", cfg.TLS.RootCAPath)
			}
			tlsConfig.RootCAs = caCertPool
		}

		if cr != nil {
			tlsConfig.GetClientCertificate = cr.GetClientCertificate
		} else if cfg.TLS.CertFilePath != "" {
			clientCert, err := tls.LoadX509KeyPair(
				filepath.Clean(cfg.TLS.CertFilePath),
				filepath.Clean(cfg.TLS.KeyFilePath),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to load kv client cert: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}

		opts.TLSConfig = tlsConfig

		// Cluster discovery (CLUSTER SLOTS / CLUSTER SHARDS) typically
		// returns IP addresses.  Build an IP→hostname reverse map from
		// the configured nodes so we can set the correct TLS ServerName
		// for each connection, even when dialing a discovered IP.
		ipToHost := buildIPToHostMap(context.Background(), cfg.Nodes)
		if len(ipToHost) > 0 {
			opts.Dialer = tlsDialer(tlsConfig, ipToHost)
		}
	}

	c := redis.NewUniversalClient(opts)
	return &redictBackend{client: c}, nil
}

func (b *redictBackend) HSet(ctx context.Context, key string, fields map[string]string) error {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return b.client.HSet(ctx, key, args...).Err()
}

func (b *redictBackend) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return b.client.HGetAll(ctx, key).Result()
}

func (b *redictBackend) Expire(ctx context.Context, key string, seconds int64) error {
	return b.client.Expire(ctx, key, time.Duration(seconds)*time.Second).Err()
}

func (b *redictBackend) Exists(ctx context.Context, key string) (bool, error) {
	n, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (b *redictBackend) HDel(ctx context.Context, key string, fields ...string) error {
	return b.client.HDel(ctx, key, fields...).Err()
}

func (b *redictBackend) Ping(ctx context.Context) error {
	return b.client.Ping(ctx).Err()
}

func (b *redictBackend) Close() {
	b.client.Close()
}
