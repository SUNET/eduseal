package types

import (
	"fmt"
)

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

// SignRequest is the request for the sign endpoint
type SignRequest struct {
	*Document
}

// MissingTokenReply is the reply when the token is missing
type MissingTokenReply struct {
	Message string `json:"message"`
}

// ErrorReply is the reply when there is an error
type ErrorReply struct {
	Message string `json:"message"`
}

func (e *ErrorReply) Error() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf("message: %q", e.Message)
}
