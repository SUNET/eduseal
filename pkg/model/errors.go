package model

import "errors"

var (
	// ErrNotFound is about
	ErrNotFound = errors.New("NOT_FOUND")

	// ErrPrivateKeyNotRSA error for private key is not rsa type
	ErrPrivateKeyNotRSA = errors.New("ERR_PRIVATE_KEY_NOT_RSA")

	// ErrPrivateKeyEmpty error for empty private key
	ErrPrivateKeyEmpty = errors.New("ERR_PRIVATE_KEY_EMPTY")

	// ErrCRTEmpty error for empty crt
	ErrCRTEmpty = errors.New("ERR_CRT_EMPTY")

	// ErrCRTNotCertificate error for wrong format of crt
	ErrCRTNotCertificate = errors.New("ERR_CRT_NOT_CERTIFICATE")

	// ErrCertificateNotValid error for not valid certificate
	ErrCertificateNotValid = errors.New("ERR_CERTIFICATE_NOT_VALID")

	// ErrInvalidDocumentType error for invalid document type
	ErrInvalidDocumentType = errors.New("ERR_INVALID_DOCUMENT_TYPE")

	// ErrNoKnownDocumentType error for no known document type
	ErrNoKnownDocumentType = errors.New("ERR_NO_KNOWN_DOCUMENT_TYPE")

	// ErrInvalidChannelName error for invalid channel name
	ErrInvalidChannelName = errors.New("ERR_INVALID_CHANNEL_NAME")

	// ErrInvalidTopicName error for invalid topic name
	ErrInvalidTopicName = errors.New("ERR_INVALID_TOPIC_NAME")
)
