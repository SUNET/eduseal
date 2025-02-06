package apiv1

import (
	"context"
	"eduseal/pkg/helpers"
	"eduseal/pkg/model"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"

	"eduseal/internal/gen/sealer/v1_sealer"
	"eduseal/internal/gen/validator/v1_validator"
)

// PDFSignRequest is the request for sign pdf
type PDFSignRequest struct {
	PDF string `json:"pdf" validate:"required,base64"`
}

// PDFSignReply is the reply for sign pdf
type PDFSignReply struct {
	Data *v1_sealer.SealReply `json:"data"`
}

// PDFSign is the request to sign pdf
//
//	@Summary		Sign pdf
//	@ID				pdf-sign
//	@Description	sign base64 encoded PDF
//	@Tags			eduseal
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	PDFSignReply			"Success"
//	@Failure		400	{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			req	body		PDFSignRequest			true	" "
//	@Router			/pdf/sign [post]
func (c *Client) PDFSign(ctx context.Context, req *PDFSignRequest) (*PDFSignReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:PDFSign")
	defer span.End()

	if req.PDF == "" {
		span.SetStatus(codes.Error, helpers.ErrEmptyPDF.Error())
		return nil, helpers.ErrEmptyPDF
	}

	transactionID := uuid.NewString()

	reply := &PDFSignReply{
		Data: &v1_sealer.SealReply{
			TransactionId: transactionID,
		},
	}

	request := &v1_sealer.SealRequest{
		Data:          req.PDF,
		TransactionId: transactionID,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to marshal request")
		return nil, err
	}

	c.log.Debug("PDFSign", "transaction_id", transactionID)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := c.stream.Seal.Publish(ctx, requestJSON, transactionID); err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to publish to stream")
		return nil, err
	}

	if err := c.kv.MetricSigning.Inc(ctx); err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to increment metric")
		return nil, err
	}

	return reply, nil
}

// PDFGetSignedRequest is the request for get signed pdf
type PDFGetSignedRequest struct {
	TransactionID string `uri:"transaction_id" binding:"required"`
}

// PDFGetSignedReply is the reply for the signed pdf
type PDFGetSignedReply struct {
	Data *model.Document `json:"data"`
}

// PDFGetSigned is the request to get signed pdfs
//
//	@Summary		fetch singed pdf
//	@ID				pdf-fetch
//	@Description	fetch a singed pdf
//	@Tags			eduseal
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	PDFGetSignedReply		"Success"
//	@Failure		400				{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			transaction_id	path		string					true	"transaction_id"
//	@Router			/pdf/{transaction_id} [get]
func (c *Client) PDFGetSigned(ctx context.Context, req *PDFGetSignedRequest) (*PDFGetSignedReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:PDFGetSigned")
	defer span.End()

	if !c.cfg.Common.Mongo.Disable {
		if c.db.EduSealSigningColl.IsRevoked(ctx, req.TransactionID) {
			span.SetStatus(codes.Error, helpers.ErrDocumentIsRevoked.Error())
			return nil, helpers.ErrDocumentIsRevoked
		}
	}

	signedDoc, err := c.kv.Doc.GetSigned(ctx, req.TransactionID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to get signed document")
		return nil, err
	}

	resp := &PDFGetSignedReply{
		Data: signedDoc,
	}

	if err := c.kv.MetricFetching.Inc(ctx); err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to increment metric")
		return nil, err
	}

	return resp, nil
}

// PDFValidateRequest is the request for verify pdf
type PDFValidateRequest struct {
	PDF string `json:"pdf"`
}

// PDFValidateReply is the reply for verify pdf
type PDFValidateReply struct {
	Data *v1_validator.ValidateReply `json:"data"`
}

// PDFValidate is the handler for verify pdf
//
//	@Summary		Validate pdf
//	@ID				pdf-validate
//	@Description	validate a signed base64 encoded PDF
//	@Tags			eduseal
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	PDFValidateReply		"Success"
//	@Failure		400	{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			req	body		PDFValidateRequest		true	" "
//	@Router			/pdf/validate [post]
func (c *Client) PDFValidate(ctx context.Context, req *PDFValidateRequest) (*PDFValidateReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:PDFValidate")
	defer span.End()

	validation, err := c.grpcClient.Validator.Validate(ctx, uuid.NewString(), req.PDF)
	if err != nil {
		return nil, err
	}

	c.log.Debug("PDFValidate", "validation", validation)

	reply := &PDFValidateReply{
		Data: validation,
	}

	if err := c.kv.MetricValidations.Inc(ctx); err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to increment metric")
		return nil, err
	}

	return reply, nil
}

// PDFRevokeRequest is the request for revoke pdf
type PDFRevokeRequest struct {
	TransactionID string `uri:"transaction_id" binding:"required"`
}

// PDFRevokeReply is the reply for revoke pdf
type PDFRevokeReply struct {
	Data struct {
		Status bool `json:"status"`
	} `json:"data"`
}

// PDFRevoke is the request to revoke pdf
//
//	@Summary		revoke signed pdf
//	@ID				pdf-revoke
//	@Description	revoke a singed pdf
//	@Tags			eduseal
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	PDFRevokeReply			"Success"
//	@Failure		400				{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			transaction_id	path		string					true	"transaction_id"
//	@Router			/pdf/revoke/{transaction_id} [put]
func (c *Client) PDFRevoke(ctx context.Context, req *PDFRevokeRequest) (*PDFRevokeReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:PDFRevoke")
	defer span.End()

	if c.cfg.Common.Mongo.Disable {
		reply := &PDFRevokeReply{
			Data: struct {
				Status bool `json:"status"`
			}{
				Status: false,
			},
		}
		return reply, nil
	}

	if err := c.db.EduSealSigningColl.Revoke(ctx, req.TransactionID); err != nil {
		return nil, err
	}

	reply := &PDFRevokeReply{
		Data: struct {
			Status bool `json:"status"`
		}{
			Status: true,
		},
	}

	return reply, nil
}
