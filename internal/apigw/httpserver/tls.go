package httpserver

import (
	"context"
	"crypto/tls"
	"eduseal/pkg/certreloader"
	"fmt"
)

func (s *Service) applyTLSConfig(ctx context.Context) error {
	reloader, err := certreloader.New(
		s.config.APIGW.APIServer.TLS.CertFilePath,
		s.config.APIGW.APIServer.TLS.KeyFilePath,
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("server cert reloader: %w", err)
	}
	s.certReloader = reloader

	s.server.TLSConfig = &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: reloader.GetCertificate,
	}
	return nil
}
