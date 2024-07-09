package model

// Document is the general document type
type Document struct {
	TransactionID string `json:"transaction_id" bson:"transaction_id" redis:"transaction_id"`
	Data          string `json:"data" bson:"base64_data" redis:"data"`
	SealerBackend string `json:"sealer_backend" bson:"sealer_backend" redis:"sealer_backend"`
	Message       string `json:"message,omitempty" bson:"message" redis:"message"`
	RevokedAt     int64  `json:"revoked_at,omitempty" bson:"revoked_at" redis:"revoke_at"`
	Reason        string `json:"reason,omitempty" bson:"reason" redis:"reason"`
}
