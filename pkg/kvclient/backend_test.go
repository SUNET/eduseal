package kvclient

import (
	"eduseal/pkg/model"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- newValkeyBackend TLS error paths ----------

func TestNewValkeyBackend_TLS_BadRootCA(t *testing.T) {
	_, err := newValkeyBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
		TLS: model.TLS{
			Enabled:      true,
			RootCAPath:   "/nonexistent/ca.crt",
			CertFilePath: "testdata/client.crt",
			KeyFilePath:  "testdata/client.key",
		},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read kv root CA")
}

func TestNewValkeyBackend_TLS_BadRootCAPEM(t *testing.T) {
	// Write a file that is not valid PEM.
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.crt")
	require.NoError(t, os.WriteFile(bad, []byte("not-a-cert"), 0o600))

	_, err := newValkeyBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
		TLS: model.TLS{
			Enabled:      true,
			RootCAPath:   bad,
			CertFilePath: "testdata/client.crt",
			KeyFilePath:  "testdata/client.key",
		},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse kv root CA")
}

func TestNewValkeyBackend_TLS_BadClientCert(t *testing.T) {
	_, err := newValkeyBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
		TLS: model.TLS{
			Enabled:      true,
			CertFilePath: "/nonexistent/client.crt",
			KeyFilePath:  "/nonexistent/client.key",
		},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load kv client cert")
}

// ---------- newRedictBackend TLS error paths ----------

func TestNewRedictBackend_TLS_BadRootCA(t *testing.T) {
	_, err := newRedictBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
		TLS: model.TLS{
			Enabled:      true,
			RootCAPath:   "/nonexistent/ca.crt",
			CertFilePath: "testdata/client.crt",
			KeyFilePath:  "testdata/client.key",
		},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read kv root CA")
}

func TestNewRedictBackend_TLS_BadRootCA_PEM(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.crt")
	require.NoError(t, os.WriteFile(bad, []byte("not-a-cert"), 0o600))

	_, err := newRedictBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
		TLS: model.TLS{
			Enabled:      true,
			RootCAPath:   bad,
			CertFilePath: "testdata/client.crt",
			KeyFilePath:  "testdata/client.key",
		},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse kv root CA")
}

func TestNewRedictBackend_TLS_BadClientCert(t *testing.T) {
	_, err := newRedictBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
		TLS: model.TLS{
			Enabled:      true,
			CertFilePath: "/nonexistent/client.crt",
			KeyFilePath:  "/nonexistent/client.key",
		},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load kv client cert")
}

// ---------- newRedictBackend without TLS (no server needed for creation) ----------

func TestNewRedictBackend_NoTLS(t *testing.T) {
	b, err := newRedictBackend(&model.KV{
		Nodes: []string{"localhost:6379"},
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, b)
	b.Close()
}

func TestNewRedictBackend_MultiNode(t *testing.T) {
	b, err := newRedictBackend(&model.KV{
		Nodes: []string{"localhost:6379", "localhost:6380"},
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, b)
	b.Close()
}
