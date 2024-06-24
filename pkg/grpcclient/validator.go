package grpcclient

import (
	"context"
	"eduseal/internal/gen/validator/v1_validator"
)

// Validator is the gRPC validator client object
type Validator struct {
	client      *Client
	scheme      string
	serviceName string
	DNS         map[string][]string
}

// Validate sends a request to the validator service to validate the signature of a PDF
func (c *Validator) Validate(ctx context.Context, transactionID, pdf string) (*v1_validator.ValidateReply, error) {
	conn, err := c.client.rrConn(ctx, c.scheme, c.serviceName)
	defer conn.Close()
	if err != nil {
		return nil, err
	}
	grpcClient := v1_validator.NewValidatorClient(conn)

	validation, err := grpcClient.Validate(ctx, &v1_validator.ValidateRequest{Pdf: pdf})
	if err != nil {
		return nil, err
	}

	return validation, nil

}
