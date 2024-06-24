package etcdclient

import (
	"context"
	"eduseal/internal/gen/sealer/v1_sealer"
	"eduseal/pkg/helpers"
	"eduseal/pkg/model"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"

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
func (d *Doc) SaveSigned(ctx context.Context, doc *v1_sealer.SealReply) error {
	ctx, span := d.client.tp.Start(ctx, "kv:SaveSigned")
	defer span.End()

	if doc.TransactionId == "" {
		span.SetStatus(codes.Error, helpers.ErrNoTransactionID.Error())
		return helpers.ErrNoTransactionID
	}

	// TTL is 1 hour
	grant, err := d.client.EtcdClient.Grant(ctx, 3600)
	if err != nil {
		return err
	}

	_, err = d.client.EtcdClient.Put(ctx, d.signedKey(doc.TransactionId), doc.Pdf, clientv3.WithLease(grant.ID))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

// ExistSigned checks if the signed document exists
func (d *Doc) ExistSigned(ctx context.Context, transactionID string) (bool, error) {
	ctx, span := d.client.tp.Start(ctx, "kv:ExistSigned")
	defer span.End()

	resp, err := d.client.EtcdClient.Get(ctx, d.signedKey(transactionID))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	return len(resp.Kvs) > 0, nil
}

// GetSigned returns the signed document and the timestamp when it was signed
func (d *Doc) GetSigned(ctx context.Context, transactionID string) (*model.Document, error) {
	ctx, span := d.client.tp.Start(ctx, "kv:GetSigned")
	defer span.End()

	ok, err := d.ExistSigned(ctx, transactionID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if !ok {
		return nil, helpers.ErrNoDocumentFound
	}

	resp, err := d.client.EtcdClient.Get(ctx, d.signedKey(transactionID))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	dest := &model.Document{
		TransactionID: transactionID,
		Base64Data:    string(resp.Kvs[0].Value),
		Message:       "",
		RevokedTS:     0,
		ModifyTS:      0,
		CreateTS:      0,
		Reason:        "",
		Location:      "",
		Name:          "",
		ContactInfo:   "",
	}
	return dest, nil
}

//
//// DelSigned deletes the signed document
//func (d *Doc) DelSigned(ctx context.Context, transactionID string) error {
//	ctx, span := d.client.tp.Start(ctx, "kv:DelSigned")
//	defer span.End()
//
//	d.client.log.Debug("Deleting signed document", "transactionID", transactionID)
//
//	return d.client.RedisClient.HDel(ctx, d.signedKey(transactionID), "base64_data", "ts").Err()
//}
//
//// AddTTLSigned marks the signed document for deletion
//func (d *Doc) AddTTLSigned(ctx context.Context, transactionID string) error {
//	ctx, span := d.client.tp.Start(ctx, "kv:AddTTLSigned")
//	defer span.End()
//
//	expTime := time.Duration(d.client.cfg.Common.ETCD.PDF.KeepSignedDuration)
//	return d.client.RedisClient.Expire(ctx, d.signedKey(transactionID), expTime*time.Second).Err()
//}
//
