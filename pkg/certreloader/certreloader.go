package certreloader

import (
	"crypto/tls"
	"eduseal/pkg/logger"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// CertReloader watches a TLS cert+key file pair on disk and atomically
// reloads them when the files change. It exposes GetCertificate and
// GetClientCertificate callbacks suitable for tls.Config.
type CertReloader struct {
	certPath string
	keyPath  string
	cert     atomic.Pointer[tls.Certificate]
	log      *logger.Log
	watcher  *fsnotify.Watcher
	done     chan struct{}
}

// New loads the initial certificate and starts watching for file changes.
func New(certPath, keyPath string, log *logger.Log) (*CertReloader, error) {
	r := &CertReloader{
		certPath: certPath,
		keyPath:  keyPath,
		log:      log,
		done:     make(chan struct{}),
	}

	if err := r.reload(); err != nil {
		return nil, fmt.Errorf("initial cert load: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify new watcher: %w", err)
	}
	r.watcher = watcher

	// Watch the directories, not the files: handles atomic rename, symlink recreation, etc.
	dirs := uniqueDirs(certPath, keyPath)
	for _, d := range dirs {
		if err := watcher.Add(d); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("fsnotify watch %q: %w", d, err)
		}
	}

	go r.watchLoop()

	r.log.Info("CertReloader started", "cert", certPath, "key", keyPath)
	return r, nil
}

// GetCertificate is a tls.Config.GetCertificate callback (server-side).
func (r *CertReloader) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert := *r.cert.Load()
	return &cert, nil
}

// GetClientCertificate is a tls.Config.GetClientCertificate callback (client-side).
func (r *CertReloader) GetClientCertificate(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	cert := *r.cert.Load()
	return &cert, nil
}

// Close stops the file watcher.
func (r *CertReloader) Close() error {
	close(r.done)
	return r.watcher.Close()
}

func (r *CertReloader) reload() error {
	cert, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return err
	}
	r.cert.Store(&cert)
	return nil
}

func (r *CertReloader) watchLoop() {
	// Debounce: after the first relevant event, wait before reloading
	// so that both cert and key files have time to be written.
	const debounce = 2 * time.Second
	var timer *time.Timer

	certBase := filepath.Base(r.certPath)
	keyBase := filepath.Base(r.keyPath)

	for {
		select {
		case <-r.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}
			base := filepath.Base(event.Name)
			if base != certBase && base != keyBase {
				continue
			}
			relevant := event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0
			if !relevant {
				continue
			}

			if timer == nil {
				timer = time.AfterFunc(debounce, r.debouncedReload)
			} else {
				timer.Reset(debounce)
			}

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			r.log.Error(err, "certreloader fsnotify error")
		}
	}
}

func (r *CertReloader) debouncedReload() {
	if err := r.reload(); err != nil {
		r.log.Error(err, "certreloader reload failed, keeping previous certificate")
		return
	}
	r.log.Info("TLS certificate reloaded", "cert", r.certPath, "key", r.keyPath)
}

func uniqueDirs(paths ...string) []string {
	seen := make(map[string]struct{})
	var dirs []string
	for _, p := range paths {
		d := filepath.Dir(p)
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			dirs = append(dirs, d)
		}
	}
	return dirs
}
