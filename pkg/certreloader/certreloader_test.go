package certreloader

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"eduseal/pkg/logger"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateSelfSigned(t *testing.T, cn string) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	return
}

func writeCertKey(t *testing.T, dir string, certPEM, keyPEM []byte) (string, string) {
	t.Helper()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	return certPath, keyPath
}

func TestInitialLoad(t *testing.T) {
	dir := t.TempDir()
	certPEM, keyPEM := generateSelfSigned(t, "initial")
	certPath, keyPath := writeCertKey(t, dir, certPEM, keyPEM)

	log := logger.NewSimple("test")
	r, err := New(certPath, keyPath, log)
	require.NoError(t, err)
	defer r.Close()

	cert, err := r.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.Equal(t, "initial", leaf.Subject.CommonName)
}

func TestReloadOnFileChange(t *testing.T) {
	dir := t.TempDir()
	certPEM1, keyPEM1 := generateSelfSigned(t, "first")
	certPath, keyPath := writeCertKey(t, dir, certPEM1, keyPEM1)

	log := logger.NewSimple("test")
	r, err := New(certPath, keyPath, log)
	require.NoError(t, err)
	defer r.Close()

	// Write a new cert+key pair
	certPEM2, keyPEM2 := generateSelfSigned(t, "second")
	require.NoError(t, os.WriteFile(certPath, certPEM2, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM2, 0600))

	// Wait for debounce + processing
	time.Sleep(4 * time.Second)

	cert, err := r.GetCertificate(nil)
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.Equal(t, "second", leaf.Subject.CommonName)
}

func TestInvalidCertKeepsOld(t *testing.T) {
	dir := t.TempDir()
	certPEM, keyPEM := generateSelfSigned(t, "good")
	certPath, keyPath := writeCertKey(t, dir, certPEM, keyPEM)

	log := logger.NewSimple("test")
	r, err := New(certPath, keyPath, log)
	require.NoError(t, err)
	defer r.Close()

	// Write garbage to the cert file
	require.NoError(t, os.WriteFile(certPath, []byte("not a cert"), 0600))

	// Wait for debounce + processing
	time.Sleep(4 * time.Second)

	cert, err := r.GetCertificate(nil)
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.Equal(t, "good", leaf.Subject.CommonName)
}

func TestGetClientCertificate(t *testing.T) {
	dir := t.TempDir()
	certPEM, keyPEM := generateSelfSigned(t, "client")
	certPath, keyPath := writeCertKey(t, dir, certPEM, keyPEM)

	log := logger.NewSimple("test")
	r, err := New(certPath, keyPath, log)
	require.NoError(t, err)
	defer r.Close()

	cert, err := r.GetClientCertificate(&tls.CertificateRequestInfo{})
	require.NoError(t, err)
	assert.NotNil(t, cert)
}
