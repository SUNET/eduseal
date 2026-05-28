package kvclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"eduseal/pkg/model"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/valkey-io/valkey-go"
)

type valkeyBackend struct {
	client valkey.Client
}

func newValkeyBackend(cfg *model.KV) (*valkeyBackend, error) {
	clientOpt := valkey.ClientOption{
		InitAddress: cfg.Nodes,
		Password:    cfg.Password,
	}

	// Single node: skip cluster discovery.
	if len(cfg.Nodes) == 1 {
		clientOpt.ForceSingleClient = true
	}

	if cfg.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.TLS.ServerName,
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

		clientCert, err := tls.LoadX509KeyPair(
			filepath.Clean(cfg.TLS.CertFilePath),
			filepath.Clean(cfg.TLS.KeyFilePath),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load kv client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}

		clientOpt.TLSConfig = tlsConfig

		// Cluster discovery (CLUSTER SLOTS / CLUSTER SHARDS) typically
		// returns IP addresses.  Build an IP→hostname reverse map from
		// the configured nodes so we can set the correct TLS ServerName
		// for each connection, even when dialing a discovered IP.
		ipToHost := buildIPToHostMap(cfg.Nodes)
		if len(ipToHost) > 0 {
			clientOpt.DialCtxFn = func(ctx context.Context, addr string, dialer *net.Dialer, tc *tls.Config) (net.Conn, error) {
				host, _, _ := net.SplitHostPort(addr)
				if hostname, ok := ipToHost[host]; ok {
					tc = tc.Clone()
					tc.ServerName = hostname
				}
				d := tls.Dialer{NetDialer: dialer, Config: tc}
				return d.DialContext(ctx, "tcp", addr)
			}
		}
	}

	c, err := valkey.NewClient(clientOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey client: %w", err)
	}
	return &valkeyBackend{client: c}, nil
}

func (b *valkeyBackend) HSet(ctx context.Context, key string, fields map[string]string) error {
	cmd := b.client.B().Hset().Key(key).FieldValue()
	for k, v := range fields {
		cmd = cmd.FieldValue(k, v)
	}
	return b.client.Do(ctx, cmd.Build()).Error()
}

func (b *valkeyBackend) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return b.client.Do(ctx, b.client.B().Hgetall().Key(key).Build()).AsStrMap()
}

func (b *valkeyBackend) Expire(ctx context.Context, key string, seconds int64) error {
	return b.client.Do(ctx, b.client.B().Expire().Key(key).Seconds(seconds).Build()).Error()
}

func (b *valkeyBackend) Exists(ctx context.Context, key string) (bool, error) {
	n, err := b.client.Do(ctx, b.client.B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (b *valkeyBackend) HDel(ctx context.Context, key string, fields ...string) error {
	cmd := b.client.B().Hdel().Key(key).Field(fields...).Build()
	return b.client.Do(ctx, cmd).Error()
}

func (b *valkeyBackend) Ping(ctx context.Context) error {
	return b.client.Do(ctx, b.client.B().Ping().Build()).Error()
}

func (b *valkeyBackend) Close() {
	b.client.Close()
}
