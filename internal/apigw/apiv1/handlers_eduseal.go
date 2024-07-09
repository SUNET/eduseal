package apiv1

import (
	"context"
	"eduseal/pkg/helpers"
	"eduseal/pkg/model"
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
	span.AddEvent("PDFSign")

	if err := helpers.Check(ctx, c.cfg, req, c.log); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	transactionID := uuid.NewString()

	c.log.Debug("PDFSign", "transaction_id", transactionID)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	signedDoc, err := c.grpcClient.Sealer.Seal(ctx, transactionID, req.PDF)
	if err != nil {
		c.log.Error(err, "gRPC request failed")
		return nil, err
	}

	reply := &PDFSignReply{
		Data: signedDoc,
	}
	c.log.Debug("PDFSign", "reply", reply)

	if err := c.kv.Doc.SaveSigned(ctx, &model.Document{
		TransactionID: signedDoc.TransactionId,
		Data:          signedDoc.Data,
		SealerBackend: signedDoc.SealerBackend,
	}); err != nil {
		span.SetStatus(codes.Error, err.Error())
		c.log.Error(err, "save seald doc failed")
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

	if err := helpers.Check(ctx, c.cfg, req, c.log); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if !c.cfg.Common.Mongo.Disable {
		if c.db.EduSealSigningColl.IsRevoked(ctx, req.TransactionID) {
			span.SetStatus(codes.Error, helpers.ErrDocumentIsRevoked.Error())
			return nil, helpers.ErrDocumentIsRevoked
		}
	}

	signedDoc, err := c.kv.Doc.GetSigned(ctx, req.TransactionID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	resp := &PDFGetSignedReply{
		Data: signedDoc,
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
