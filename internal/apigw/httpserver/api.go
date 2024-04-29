package httpserver

import (
	"context"
	"eduseal/internal/apigw/apiv1"
	apiv1_status "eduseal/internal/gen/status/apiv1.status"
)

// Apiv1 interface
type Apiv1 interface {
	// eduSeal endpoints
	PDFSign(ctx context.Context, req *apiv1.PDFSignRequest) (*apiv1.PDFSignReply, error)
	PDFValidate(ctx context.Context, req *apiv1.PDFValidateRequest) (*apiv1.PDFValidateReply, error)
	PDFGetSigned(ctx context.Context, req *apiv1.PDFGetSignedRequest) (*apiv1.PDFGetSignedReply, error)
	PDFRevoke(ctx context.Context, req *apiv1.PDFRevokeRequest) (*apiv1.PDFRevokeReply, error)

	// misc endpoints
	Health(ctx context.Context, req *apiv1_status.StatusRequest) (*apiv1_status.StatusReply, error)
}
