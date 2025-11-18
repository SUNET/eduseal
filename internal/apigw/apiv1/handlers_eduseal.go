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
//	@Summary		Seal pdf
//	@ID				pdf-seal
//	@Description	seal base64 encoded PDF
//	@Tags			EduSeal
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	PDFSignReply			"Success"
//	@Failure		400	{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			req	body		PDFSignRequest			true	" "
//	@Router			/pdf/sign [post]
func (c *Client) PDFSign(ctx context.Context, req *PDFSignRequest) (*PDFSignReply, error) {
	ctx, span := c.tracer.Start(ctx, "apiv1:PDFSign")
	defer span.End()

	counter, err := c.metric.Int64Counter("pdf_seal_counter")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to create pdf seal counter")
		return nil, err
	}

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

	counter.Add(ctx, 1)

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

// PDFGetSigned is the request to get sealed pdf
//
//	@Summary		fetch sealed pdf
//	@ID				pdf-fetch
//	@Description	fetch a sealed pdf
//	@Tags			EduSeal
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	PDFGetSignedReply		"Success"
//	@Failure		400				{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			transaction_id	path		string					true	"transaction_id"
//	@Router			/pdf/{transaction_id} [get]
func (c *Client) PDFGetSigned(ctx context.Context, req *PDFGetSignedRequest) (*PDFGetSignedReply, error) {
	ctx, span := c.tracer.Start(ctx, "apiv1:PDFGetSigned")
	defer span.End()

	counter, err := c.metric.Int64Counter("pdf_get_sealed_counter")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to create pdf get sealed counter")
		return nil, err
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

	counter.Add(ctx, 1)

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
//	@Description	validate a sealed base64 encoded PDF
//	@Tags			EduSeal
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	PDFValidateReply		"Success"
//	@Failure		400	{object}	helpers.ErrorResponse	"Bad Request"
//	@Param			req	body		PDFValidateRequest		true	" "
//	@Router			/pdf/validate [post]
func (c *Client) PDFValidate(ctx context.Context, req *PDFValidateRequest) (*PDFValidateReply, error) {
	ctx, span := c.tracer.Start(ctx, "apiv1:PDFValidate")
	defer span.End()

	counter, err := c.metric.Int64Counter("pdf_validation_counter")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "failed to create pdf validation counter")
		return nil, err
	}

	validation, err := c.grpcClient.Validator.Validate(ctx, uuid.NewString(), req.PDF)
	if err != nil {
		return nil, err
	}

	c.log.Debug("PDFValidate", "validation", validation)

	reply := &PDFValidateReply{
		Data: validation,
	}

	counter.Add(ctx, 1)

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
