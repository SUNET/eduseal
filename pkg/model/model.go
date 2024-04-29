package model

import "time"

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
