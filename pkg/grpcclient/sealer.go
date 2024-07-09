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
func (c *Sealer) Seal(ctx context.Context, transactionID, data string) (*v1_sealer.SealReply, error) {
	conn, err := c.client.rrConn(ctx, c.scheme, c.client.cfg.Common.SealerServiceName)
	defer conn.Close()
	if err != nil {
		c.client.log.Error(err, "failed to connect to sealer")
		return nil, err
	}

	grpcClient := v1_sealer.NewSealerClient(conn)

	seal, err := grpcClient.Seal(ctx, &v1_sealer.SealRequest{TransactionId: transactionID, Data: data})
	if err != nil {
		c.client.log.Error(err, "failed to send call to sealer")
		return nil, err
	}

	return seal, nil

}
