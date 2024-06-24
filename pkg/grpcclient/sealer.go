package grpcclient

import (
	"context"
	"eduseal/internal/gen/sealer/v1_sealer"
)

// Sealer is the gRPC sealer client object
type Sealer struct {
	client      *Client
	scheme      string
	serviceName string
	DNS         map[string][]string
}

// Seal sends a request to the sealer service to seal a PDF
func (c *Sealer) Seal(ctx context.Context, transactionID, pdf string) (*v1_sealer.SealReply, error) {
	conn, err := c.client.rrConn(ctx, c.scheme, c.serviceName)
	defer conn.Close()
	if err != nil {
		return nil, err
	}

	grpcClient := v1_sealer.NewSealerClient(conn)

	seal, err := grpcClient.Seal(ctx, &v1_sealer.SealRequest{TransactionId: transactionID, Pdf: pdf})
	if err != nil {
		return nil, err
	}

	return seal, nil

}
