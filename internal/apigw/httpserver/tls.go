package httpserver

import (
	"context"
	"crypto/tls"
)

func (s *Service) applyTLSConfig(ctx context.Context) {
	cfg := &tls.Config{
		//MinVersion: tls.VersionTLS12,
		//CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		//PreferServerCipherSuites: true,
	}

	s.server.TLSConfig = cfg
}
