package kvclient

import (
	"context"
	"eduseal/pkg/helpers"
	"eduseal/pkg/model"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/codes"
)

// Doc holds the document kv object
type Doc struct {
	client *Client
	key    string
}

func (d Doc) mkKey(transactionID, docType string) string {
	return fmt.Sprintf(d.key, transactionID, docType)
}

func (d Doc) signedKey(transactionID string) string {
	return d.mkKey(transactionID, "signed")
}

// SaveSigned saves the signed document and the timestamp when it was signed
func (d *Doc) SaveSigned(ctx context.Context, doc *model.Document) error {
	ctx, span := d.client.tp.Start(ctx, "kv:SaveSigned")
	defer span.End()

	if doc.TransactionID == "" {
		span.SetStatus(codes.Error, helpers.ErrNoTransactionID.Error())
		return helpers.ErrNoTransactionID
	}

	key := d.signedKey(doc.TransactionID)
	b := d.client.backend

	fields := map[string]string{
		"transaction_id": doc.TransactionID,
		"data":           doc.Data,
		"sealer_backend": doc.SealerBackend,
		"message":        doc.Message,
		"revoke_at":      strconv.FormatInt(doc.RevokedAt, 10),
		"reason":         doc.Reason,
	}

	if err := b.HSet(ctx, key, fields); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := b.Expire(ctx, key, int64(10*time.Second/time.Second)); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// GetSigned returns the signed document and the timestamp when it was signed
func (d *Doc) GetSigned(ctx context.Context, transactionID string) (*model.Document, error) {
	ctx, span := d.client.tp.Start(ctx, "kv:GetSigned")
	defer span.End()

	b := d.client.backend
	result, err := b.HGetAll(ctx, d.signedKey(transactionID))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	revokedAt, _ := strconv.ParseInt(result["revoke_at"], 10, 64)
	dest := &model.Document{
		TransactionID: result["transaction_id"],
		Data:          result["data"],
		SealerBackend: result["sealer_backend"],
		Message:       result["message"],
		RevokedAt:     revokedAt,
		Reason:        result["reason"],
	}
	return dest, nil
}

// ExistsSigned returns true if the signed document exists
func (d *Doc) ExistsSigned(ctx context.Context, transactionID string) bool {
	ctx, span := d.client.tp.Start(ctx, "kv:ExistsSigned")
	defer span.End()

	b := d.client.backend
	exists, err := b.Exists(ctx, d.signedKey(transactionID))
	if err != nil {
		return false
	}
	return exists
}

// DelSigned deletes the signed document
func (d *Doc) DelSigned(ctx context.Context, transactionID string) error {
	ctx, span := d.client.tp.Start(ctx, "kv:DelSigned")
	defer span.End()

	d.client.log.Debug("Deleting signed document", "transactionID", transactionID)

	return d.client.backend.HDel(ctx, d.signedKey(transactionID), "base64_data", "ts")
}
