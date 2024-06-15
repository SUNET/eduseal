package model

import (
	"time"
)

// Meta is a generic type for metadata
type Meta struct {
	UploadID     string    `json:"upload_id" bson:"upload_id"`
	Timestamp    time.Time `json:"timestamp" bson:"timestamp"`
	DocumentType string    `json:"document_type"`
}

// Response is a generic type for responses
type Response struct {
	Data  any `json:"data"`
	Error any `json:"error"`
}

// Document is the general document type
type Document struct {
	TransactionID string `json:"transaction_id" bson:"transaction_id" redis:"transaction_id"`
	Base64Data    string `json:"base64_data" bson:"base64_data" redis:"base64_data"`
	Error         string `json:"error,omitempty" bson:"error" redis:"error"`
	Message       string `json:"message,omitempty" bson:"-" redis:"-"`
	RevokedTS     int64  `json:"revoked_ts,omitempty" bson:"revoked_ts" redis:"revoke_ts"`
	ModifyTS      int64  `json:"modify_ts,omitempty" bson:"modify_ts" redis:"modify_ts"`
	CreateTS      int64  `json:"create_ts,omitempty" mongo:"create_ts" redis:"create_ts"`
	Reason        string `json:"reason,omitempty"`
	Location      string `json:"location,omitempty"`
	Name          string `json:"name,omitempty"`
	ContactInfo   string `json:"contact_info,omitempty"`
}

// Validation is the reply for the validate endpoint
type Validation struct {
	ValidSignature bool   `json:"valid_signature"`
	TransactionID  string `json:"transaction_id"`
	Message        string `json:"message"`
	IsRevoked      bool   `json:"is_revoked"`
	Error          string `json:"error,omitempty"`
}
