package milvus

import (
	"errors"
	"fmt"
)

// Error types for better error handling
var (
	ErrCollectionNameRequired = errors.New("collection name required")
	ErrEmptyData              = errors.New("no valid columns provided")
	ErrEmptyVectorArray       = errors.New("empty vector array")
	ErrNoSearchRequests       = errors.New("at least one search request required")
	ErrInvalidDataType        = errors.New("invalid data type")
	ErrUnsupportedType        = errors.New("unsupported type")
	ErrSchemaParseError       = errors.New("failed to parse schema")
)

// MilvusError wraps errors with additional context
type MilvusError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Context string // additional context
}

func (e *MilvusError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Context, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *MilvusError) Unwrap() error {
	return e.Err
}

// newError creates a new MilvusError
func newError(op string, err error, context string) error {
	return &MilvusError{
		Op:      op,
		Err:     err,
		Context: context,
	}
}

// wrapError wraps an error with operation context
func wrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &MilvusError{
		Op:  op,
		Err: err,
	}
}
