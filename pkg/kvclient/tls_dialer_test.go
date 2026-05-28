package kvclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIPToHostMap_Hostnames(t *testing.T) {
	// Use localhost which always resolves to 127.0.0.1.
	m := buildIPToHostMap([]string{"localhost:6379", "localhost:6380"})

	assert.Contains(t, m, "127.0.0.1:6379")
	assert.Contains(t, m, "127.0.0.1:6380")
	assert.Equal(t, "localhost", m["127.0.0.1:6379"])
	assert.Equal(t, "localhost", m["127.0.0.1:6380"])
}

func TestBuildIPToHostMap_IPsAreSkipped(t *testing.T) {
	m := buildIPToHostMap([]string{"10.0.0.1:6379", "192.168.1.1:6380"})
	assert.Empty(t, m, "bare IPs should not appear in the map")
}

func TestBuildIPToHostMap_Empty(t *testing.T) {
	m := buildIPToHostMap(nil)
	assert.Empty(t, m)

	m = buildIPToHostMap([]string{})
	assert.Empty(t, m)
}

func TestBuildIPToHostMap_BadAddresses(t *testing.T) {
	m := buildIPToHostMap([]string{
		"no-port",
		":6379",
		"",
	})
	assert.Empty(t, m)
}

func TestBuildIPToHostMap_UnresolvableHost(t *testing.T) {
	m := buildIPToHostMap([]string{"this-host-does-not-exist.invalid:6379"})
	assert.Empty(t, m)
}

func TestBuildIPToHostMap_DuplicateHostDifferentPorts(t *testing.T) {
	m := buildIPToHostMap([]string{
		"localhost:6379",
		"localhost:6380",
		"localhost:6381",
	})
	// All three ports on the same IP should be present.
	assert.Equal(t, "localhost", m["127.0.0.1:6379"])
	assert.Equal(t, "localhost", m["127.0.0.1:6380"])
	assert.Equal(t, "localhost", m["127.0.0.1:6381"])
}

// ---------- TLS dialer tests ----------

// selfSignedCert creates a throwaway CA + leaf cert with DNS SANs only
// (no IP SANs), matching production certificates.
// Returns the leaf tls.Certificate and a CA pool that trusts it.
func selfSignedCert(t *testing.T, dnsNames ...string) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	cn := "localhost"
	if len(dnsNames) > 0 {
		cn = dnsNames[0]
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     dnsNames,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	require.NoError(t, err)

	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER}),
	)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	return cert, pool
}

// tlsServer starts a TLS listener on 127.0.0.1:0, returns the listener.
// A background goroutine accepts one connection, completes the handshake,
// then closes it.
func tlsServer(t *testing.T, cert tls.Certificate) net.Listener {
	t.Helper()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Force handshake completion on the server side.
		if tc, ok := conn.(*tls.Conn); ok {
			_ = tc.Handshake()
		}
	}()
	return ln
}

func TestTLSDialer_SetsServerNameFromMap(t *testing.T) {
	serverCert, caPool := selfSignedCert(t, "my-node.example.com")
	ln := tlsServer(t, serverCert)

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	ipToHost := map[string]string{
		"127.0.0.1:" + port: "my-node.example.com",
	}
	baseCfg := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS12,
	}

	dial := tlsDialer(baseCfg, ipToHost)
	conn, err := dial(context.Background(), "tcp", "127.0.0.1:"+port)
	require.NoError(t, err, "dialer should set ServerName from the map and TLS should succeed")
	conn.Close()
}

func TestTLSDialer_NoMatchFailsWithDNSOnlyCert(t *testing.T) {
	// Cert has only a DNS SAN — no IP SAN, just like production.
	// When the map has no entry for the dialed IP, no ServerName is set
	// and TLS verification fails against the bare IP.
	serverCert, caPool := selfSignedCert(t, "my-node.example.com")
	ln := tlsServer(t, serverCert)

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	emptyMap := map[string]string{}
	baseCfg := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS12,
	}

	dial := tlsDialer(baseCfg, emptyMap)
	_, err := dial(context.Background(), "tcp", "127.0.0.1:"+port)
	require.Error(t, err, "without IP SAN and no map match, TLS should fail — this is the problem the dialer solves")
}

func TestTLSDialer_WrongServerNameFails(t *testing.T) {
	serverCert, caPool := selfSignedCert(t, "correct-name.example.com")
	ln := tlsServer(t, serverCert)

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	// Map to the wrong hostname — TLS verification should fail.
	ipToHost := map[string]string{
		"127.0.0.1:" + port: "wrong-name.example.com",
	}
	baseCfg := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS12,
	}

	dial := tlsDialer(baseCfg, ipToHost)
	_, err := dial(context.Background(), "tcp", "127.0.0.1:"+port)
	require.Error(t, err, "TLS should fail when ServerName doesn't match the certificate")
}
