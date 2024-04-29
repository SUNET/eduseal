package retask

import "errors"

var (
	// ErrNoResult is returned when no result is available from the queue
	ErrNoResult = errors.New("NoResultFromQueue")
)
