package httpserver

import (
	"context"
	"eduseal/internal/apigw/apiv1"
	"eduseal/internal/gen/status/v1_status"
)

// Apiv1 interface
type Apiv1 interface {
	// eduSeal endpoints
	PDFSign(ctx context.Context, req *apiv1.PDFSignRequest) (*apiv1.PDFSignReply, error)
	PDFValidate(ctx context.Context, req *apiv1.PDFValidateRequest) (*apiv1.PDFValidateReply, error)
	PDFGetSigned(ctx context.Context, req *apiv1.PDFGetSignedRequest) (*apiv1.PDFGetSignedReply, error)
	PDFRevoke(ctx context.Context, req *apiv1.PDFRevokeRequest) (*apiv1.PDFRevokeReply, error)

	// misc endpoints
	Health(ctx context.Context, req *v1_status.StatusRequest) (*v1_status.StatusReply, error)
}
