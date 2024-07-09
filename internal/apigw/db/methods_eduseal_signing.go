package db

import (
	"context"
	"eduseal/pkg/model"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/codes"
)

// EduSealSigningColl is the generic collection
type EduSealSigningColl struct {
	service *Service
	coll    *mongo.Collection
}

func (c *EduSealSigningColl) createIndex(ctx context.Context) error {
	ctx, span := c.service.tp.Start(ctx, "db:doc:createIndex")
	defer span.End()

	indexModel := mongo.IndexModel{
		Keys: bson.M{"transaction_id": 1},
	}
	_, err := c.coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// Save saves one document
func (c *EduSealSigningColl) Save(ctx context.Context, doc *model.Document) error {
	ctx, span := c.service.tp.Start(ctx, "db:doc:save")
	defer span.End()

	_, err := c.coll.InsertOne(ctx, doc)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	c.service.log.Info("saved document", "transaction_id", doc.TransactionID)
	return nil
}

// Revoke revokes a document
func (c *EduSealSigningColl) Revoke(ctx context.Context, transactionID string) error {
	ctx, span := c.service.tp.Start(ctx, "db:doc:revoke")
	defer span.End()

	filter := bson.M{
		"transaction_id": bson.M{"$eq": transactionID},
	}
	update := bson.M{
		"$set": bson.M{
			"revoked_ts": time.Now().Unix(),
		},
	}
	_, err := c.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

// IsRevoked checks if a document is revoked
func (c *EduSealSigningColl) IsRevoked(ctx context.Context, transactionID string) bool {
	ctx, span := c.service.tp.Start(ctx, "db:doc:isRevoked")
	defer span.End()

	doc, err := c.Get(ctx, transactionID)
	if err != nil {
		span.SetStatus(codes.Ok, "document not found")
		return false
	}

	if doc.RevokedAt != 0 {
		return true
	}
	return false
}

// Get gets one document
func (c *EduSealSigningColl) Get(ctx context.Context, transactionID string) (*model.Document, error) {
	ctx, span := c.service.tp.Start(ctx, "db:doc:get")
	defer span.End()

	reply := &model.Document{}
	filter := bson.M{
		"transaction_id": bson.M{"$eq": transactionID},
	}
	err := c.coll.FindOne(ctx, filter).Decode(reply)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			span.SetStatus(codes.Ok, "document not found")
			return nil, errors.New("no document found")
		}
		return nil, err
	}
	return reply, nil
}
