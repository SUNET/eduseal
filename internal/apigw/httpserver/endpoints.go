package httpserver

import (
	"context"
	"eduseal/internal/apigw/apiv1"
	"eduseal/internal/gen/status/v1_status"

	"go.opentelemetry.io/otel/codes"

	"github.com/gin-gonic/gin"
)

func (s *Service) endpointHealth(ctx context.Context, c *gin.Context) (any, error) {
	ctx, span := s.tp.Start(ctx, "httpserver:endpointHealth")
	defer span.End()

	request := &v1_status.StatusRequest{}
	reply, err := s.apiv1.Health(ctx, request)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// endpointSignPDF signs a PDF EduSeal
func (s *Service) endpointSignPDF(ctx context.Context, c *gin.Context) (interface{}, error) {
	ctx, span := s.tp.Start(ctx, "httpserver:endpointSignPDF")
	defer span.End()

	request := &apiv1.PDFSignRequest{}
	if err := s.bindV2(ctx, c, request); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	reply, err := s.apiv1.PDFSign(ctx, request)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return reply, nil
}

// endpointValidatePDF validates a signed PDF EduSeal
func (s *Service) endpointValidatePDF(ctx context.Context, c *gin.Context) (interface{}, error) {
	ctx, span := s.tp.Start(ctx, "httpserver:endpointValidatePDF")
	defer span.End()

	request := &apiv1.PDFValidateRequest{}
	if err := s.bindRequest(ctx, c, request); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	reply, err := s.apiv1.PDFValidate(ctx, request)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return reply, nil
}

// endpointGetSignedPDF returns a signed PDF EduSeal
func (s *Service) endpointGetSignedPDF(ctx context.Context, c *gin.Context) (interface{}, error) {
	ctx, span := s.tp.Start(ctx, "httpserver:endpointGetSignedPDF")
	defer span.End()

	request := &apiv1.PDFGetSignedRequest{}
	if err := s.bindRequest(ctx, c, request); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	reply, err := s.apiv1.PDFGetSigned(ctx, request)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return reply, nil
}

// endpointPDFRevoke revokes a signed PDF EduSeal
func (s *Service) endpointPDFRevoke(ctx context.Context, c *gin.Context) (any, error) {
	ctx, span := s.tp.Start(ctx, "httpserver:endpointPDFRevoke")
	defer span.End()

	request := &apiv1.PDFRevokeRequest{}
	if err := s.bindRequest(ctx, c, request); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	reply, err := s.apiv1.PDFRevoke(ctx, request)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return reply, nil
}
