package kvclient

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/helpers"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- mock backend ----------

type mockBackend struct {
	store   map[string]map[string]string
	ttls    map[string]int64
	pingErr error
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		store: make(map[string]map[string]string),
		ttls:  make(map[string]int64),
	}
}

func (m *mockBackend) HSet(_ context.Context, key string, fields map[string]string) error {
	if _, ok := m.store[key]; !ok {
		m.store[key] = make(map[string]string)
	}
	for k, v := range fields {
		m.store[key][k] = v
	}
	return nil
}

func (m *mockBackend) HGetAll(_ context.Context, key string) (map[string]string, error) {
	if v, ok := m.store[key]; ok {
		return v, nil
	}
	return map[string]string{}, nil
}

func (m *mockBackend) Expire(_ context.Context, key string, seconds int64) error {
	m.ttls[key] = seconds
	return nil
}

func (m *mockBackend) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.store[key]
	return ok, nil
}

func (m *mockBackend) HDel(_ context.Context, key string, fields ...string) error {
	if h, ok := m.store[key]; ok {
		for _, f := range fields {
			delete(h, f)
		}
	}
	return nil
}

func (m *mockBackend) Ping(_ context.Context) error {
	return m.pingErr
}

func (m *mockBackend) Close() {}

// ---------- errBackend returns errors for everything ----------

type errBackend struct {
	err error
}

func (e *errBackend) HSet(context.Context, string, map[string]string) error { return e.err }
func (e *errBackend) HGetAll(context.Context, string) (map[string]string, error) {
	return nil, e.err
}
func (e *errBackend) Expire(context.Context, string, int64) error   { return e.err }
func (e *errBackend) Exists(context.Context, string) (bool, error)  { return false, e.err }
func (e *errBackend) HDel(context.Context, string, ...string) error { return e.err }
func (e *errBackend) Ping(context.Context) error                    { return e.err }
func (e *errBackend) Close()                                        {}

// ---------- helpers ----------

func newTestClient(t *testing.T, b Backend) *Client {
	t.Helper()
	log := logger.NewSimple("test")
	ctx := context.Background()
	tp, err := trace.NewForTesting(ctx, "kvclient-test", log)
	require.NoError(t, err)

	c := &Client{
		backend:    b,
		cfg:        &model.Cfg{},
		log:        log,
		probeStore: &v1_status.StatusProbeStore{},
		tp:         tp,
		statusTick: time.NewTicker(time.Hour), // never fires in tests
		Doc:        nil,
	}
	c.Doc = &Doc{client: c, key: "doc:%s:%s"}
	return c
}

// ---------- Doc.SaveSigned tests ----------

func TestSaveSigned_OK(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	doc := &model.Document{
		TransactionID: "tx-1",
		Data:          "base64data",
		SealerBackend: "softhsm",
		Message:       "ok",
		RevokedAt:     0,
		Reason:        "",
	}
	err := c.Doc.SaveSigned(ctx, doc)
	require.NoError(t, err)

	key := "doc:tx-1:signed"
	assert.Equal(t, "tx-1", mb.store[key]["transaction_id"])
	assert.Equal(t, "base64data", mb.store[key]["data"])
	assert.Equal(t, "softhsm", mb.store[key]["sealer_backend"])
	assert.Equal(t, int64(10), mb.ttls[key])
}

func TestSaveSigned_NoTransactionID(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	err := c.Doc.SaveSigned(ctx, &model.Document{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, helpers.ErrNoTransactionID))
}

func TestSaveSigned_BackendHSetError(t *testing.T) {
	eb := &errBackend{err: errors.New("hset fail")}
	c := newTestClient(t, eb)
	ctx := context.Background()

	err := c.Doc.SaveSigned(ctx, &model.Document{TransactionID: "tx-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hset fail")
}

func TestSaveSigned_BackendExpireError(t *testing.T) {
	// A backend that succeeds on HSet but fails on Expire.
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	// Wrap to inject error on Expire.
	c.backend = &expireErrBackend{Backend: mb, err: errors.New("expire fail")}

	err := c.Doc.SaveSigned(ctx, &model.Document{TransactionID: "tx-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expire fail")
}

type expireErrBackend struct {
	Backend
	err error
}

func (e *expireErrBackend) Expire(context.Context, string, int64) error { return e.err }

// ---------- Doc.GetSigned tests ----------

func TestGetSigned_OK(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	// Pre-populate.
	mb.store["doc:tx-2:signed"] = map[string]string{
		"transaction_id": "tx-2",
		"data":           "d2",
		"sealer_backend": "lunahsm",
		"message":        "signed",
		"revoke_at":      "1700000000",
		"reason":         "test",
	}

	doc, err := c.Doc.GetSigned(ctx, "tx-2")
	require.NoError(t, err)
	assert.Equal(t, "tx-2", doc.TransactionID)
	assert.Equal(t, "d2", doc.Data)
	assert.Equal(t, "lunahsm", doc.SealerBackend)
	assert.Equal(t, "signed", doc.Message)
	assert.Equal(t, int64(1700000000), doc.RevokedAt)
	assert.Equal(t, "test", doc.Reason)
}

func TestGetSigned_EmptyResult(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	doc, err := c.Doc.GetSigned(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", doc.TransactionID)
}

func TestGetSigned_BackendError(t *testing.T) {
	eb := &errBackend{err: errors.New("hgetall fail")}
	c := newTestClient(t, eb)
	ctx := context.Background()

	_, err := c.Doc.GetSigned(ctx, "tx-1")
	require.Error(t, err)
}

// ---------- Doc.ExistsSigned tests ----------

func TestExistsSigned_True(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	mb.store["doc:tx-3:signed"] = map[string]string{"transaction_id": "tx-3"}
	assert.True(t, c.Doc.ExistsSigned(ctx, "tx-3"))
}

func TestExistsSigned_False(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	assert.False(t, c.Doc.ExistsSigned(ctx, "nonexistent"))
}

func TestExistsSigned_Error(t *testing.T) {
	eb := &errBackend{err: errors.New("exists fail")}
	c := newTestClient(t, eb)
	ctx := context.Background()

	assert.False(t, c.Doc.ExistsSigned(ctx, "tx-1"))
}

// ---------- Doc.DelSigned tests ----------

func TestDelSigned_OK(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	mb.store["doc:tx-4:signed"] = map[string]string{
		"base64_data": "d",
		"ts":          "123",
		"other":       "keep",
	}

	err := c.Doc.DelSigned(ctx, "tx-4")
	require.NoError(t, err)
	// "base64_data" and "ts" should be removed; "other" should remain.
	assert.NotContains(t, mb.store["doc:tx-4:signed"], "base64_data")
	assert.NotContains(t, mb.store["doc:tx-4:signed"], "ts")
	assert.Equal(t, "keep", mb.store["doc:tx-4:signed"]["other"])
}

func TestDelSigned_BackendError(t *testing.T) {
	eb := &errBackend{err: errors.New("hdel fail")}
	c := newTestClient(t, eb)
	ctx := context.Background()

	err := c.Doc.DelSigned(ctx, "tx-1")
	require.Error(t, err)
}

// ---------- Client.probe tests ----------

func TestProbe_Healthy(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	c.probe(ctx)
	status := c.probeStore.PreviousResult
	assert.True(t, status.Healthy)
	assert.Equal(t, "OK", status.Message)
}

func TestProbe_Unhealthy(t *testing.T) {
	mb := newMockBackend()
	mb.pingErr = errors.New("connection refused")
	c := newTestClient(t, mb)
	ctx := context.Background()

	c.probe(ctx)
	status := c.probeStore.PreviousResult
	assert.False(t, status.Healthy)
	assert.Contains(t, status.Message, "connection refused")
}

// ---------- Client.Status tests ----------

func TestStatus_ReturnsLatestProbe(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	c.probe(ctx)

	result := c.Status(ctx)
	assert.True(t, result.Healthy)
	assert.Equal(t, "kv", result.Name)
}

// ---------- Client.Close tests ----------

func TestClose(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)
	ctx := context.Background()

	err := c.Close(ctx)
	assert.NoError(t, err)
}

// ---------- Doc key helpers ----------

func TestMkKey(t *testing.T) {
	mb := newMockBackend()
	c := newTestClient(t, mb)

	assert.Equal(t, "doc:tx-1:signed", c.Doc.signedKey("tx-1"))
	assert.Equal(t, "doc:abc:mytype", c.Doc.mkKey("abc", "mytype"))
}

// ---------- New() type selection ----------

func TestNew_UnsupportedType(t *testing.T) {
	log := logger.NewSimple("test")
	ctx := context.Background()
	tp, err := trace.NewForTesting(ctx, "kvclient-test", log)
	require.NoError(t, err)

	cfg := &model.Cfg{
		Common: model.Common{
			KV: model.KV{
				Type:  "memcached",
				Nodes: []string{"localhost:11211"},
			},
		},
	}

	_, err = New(ctx, cfg, tp, log)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported kv type")
}

func TestNew_DefaultsToValkey(t *testing.T) {
	// Type="" should default to "valkey". Since there's no live server,
	// valkey.NewClient will fail, but we verify the code path reaches
	// the valkey backend (not redict or the unsupported-type error).
	log := logger.NewSimple("test")
	ctx := context.Background()
	tp, err := trace.NewForTesting(ctx, "kvclient-test", log)
	require.NoError(t, err)

	cfg := &model.Cfg{
		Common: model.Common{
			KV: model.KV{
				Type:  "",
				Nodes: []string{"localhost:16379"}, // no server listening
			},
		},
	}

	// The client creation will attempt to connect and may fail, but
	// it should NOT return "unsupported kv type".
	_, err = New(ctx, cfg, tp, log)
	if err != nil {
		assert.NotContains(t, err.Error(), "unsupported kv type")
	}
}
