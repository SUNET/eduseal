package kvclient

import (
	"context"
	"eduseal/pkg/helpers"
	"eduseal/pkg/model"
	"fmt"
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

	if err := d.client.RedictCC.Expire(ctx, d.signedKey(doc.TransactionID), 1*time.Hour).Err(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := d.client.RedictCC.HSet(ctx, d.signedKey(doc.TransactionID), doc).Err(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

// GetSigned returns the signed document and the timestamp when it was signed
func (d *Doc) GetSigned(ctx context.Context, transactionID string) (*model.Document, error) {
	ctx, span := d.client.tp.Start(ctx, "kv:GetSigned")
	defer span.End()

	dest := &model.Document{}
	if err := d.client.RedictCC.HGetAll(ctx, d.signedKey(transactionID)).Scan(dest); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return dest, nil
}

// ExistsSigned returns true if the signed document exists
func (d *Doc) ExistsSigned(ctx context.Context, transactionID string) bool {
	ctx, span := d.client.tp.Start(ctx, "kv:ExistsSigned")
	defer span.End()

	return d.client.RedictCC.Exists(ctx, d.signedKey(transactionID)).Val() == 1
}

// DelSigned deletes the signed document
func (d *Doc) DelSigned(ctx context.Context, transactionID string) error {
	ctx, span := d.client.tp.Start(ctx, "kv:DelSigned")
	defer span.End()

	d.client.log.Debug("Deleting signed document", "transactionID", transactionID)

	return d.client.RedictCC.HDel(ctx, d.signedKey(transactionID), "base64_data", "ts").Err()
}
