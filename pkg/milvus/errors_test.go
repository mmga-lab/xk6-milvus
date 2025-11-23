package milvus

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrCollectionNameRequired",
			err:  ErrCollectionNameRequired,
			msg:  "collection name required",
		},
		{
			name: "ErrEmptyData",
			err:  ErrEmptyData,
			msg:  "no valid columns provided",
		},
		{
			name: "ErrEmptyVectorArray",
			err:  ErrEmptyVectorArray,
			msg:  "empty vector array",
		},
		{
			name: "ErrNoSearchRequests",
			err:  ErrNoSearchRequests,
			msg:  "at least one search request required",
		},
		{
			name: "ErrInvalidDataType",
			err:  ErrInvalidDataType,
			msg:  "invalid data type",
		},
		{
			name: "ErrUnsupportedType",
			err:  ErrUnsupportedType,
			msg:  "unsupported type",
		},
		{
			name: "ErrSchemaParseError",
			err:  ErrSchemaParseError,
			msg:  "failed to parse schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.Equal(t, tt.msg, tt.err.Error())
		})
	}
}

func TestMilvusError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *MilvusError
		want    string
	}{
		{
			name: "with context",
			err: &MilvusError{
				Op:      "Insert",
				Err:     errors.New("connection failed"),
				Context: "failed to connect to server",
			},
			want: "Insert: failed to connect to server: connection failed",
		},
		{
			name: "without context",
			err: &MilvusError{
				Op:      "Search",
				Err:     errors.New("invalid parameter"),
				Context: "",
			},
			want: "Search: invalid parameter",
		},
		{
			name: "with empty operation",
			err: &MilvusError{
				Op:      "",
				Err:     errors.New("unknown error"),
				Context: "",
			},
			want: ": unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMilvusError_Unwrap(t *testing.T) {
	t.Run("unwrap underlying error", func(t *testing.T) {
		underlyingErr := errors.New("original error")
		milvusErr := &MilvusError{
			Op:  "Test",
			Err: underlyingErr,
		}

		unwrapped := milvusErr.Unwrap()
		assert.Equal(t, underlyingErr, unwrapped)
		assert.True(t, errors.Is(milvusErr, underlyingErr))
	})

	t.Run("unwrap predefined error", func(t *testing.T) {
		milvusErr := &MilvusError{
			Op:  "CreateCollection",
			Err: ErrCollectionNameRequired,
		}

		unwrapped := milvusErr.Unwrap()
		assert.Equal(t, ErrCollectionNameRequired, unwrapped)
		assert.True(t, errors.Is(milvusErr, ErrCollectionNameRequired))
	})

	t.Run("unwrap nil error", func(t *testing.T) {
		milvusErr := &MilvusError{
			Op:  "Test",
			Err: nil,
		}

		unwrapped := milvusErr.Unwrap()
		assert.Nil(t, unwrapped)
	})
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		err     error
		context string
		want    string
	}{
		{
			name:    "with all fields",
			op:      "Insert",
			err:     errors.New("network error"),
			context: "timeout after 30s",
			want:    "Insert: timeout after 30s: network error",
		},
		{
			name:    "without context",
			op:      "Search",
			err:     errors.New("not found"),
			context: "",
			want:    "Search: not found",
		},
		{
			name:    "with predefined error",
			op:      "Delete",
			err:     ErrCollectionNameRequired,
			context: "collection parameter missing",
			want:    "Delete: collection parameter missing: collection name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newError(tt.op, tt.err, tt.context)
			require.Error(t, err)
			assert.Equal(t, tt.want, err.Error())

			// Verify it's a MilvusError
			var milvusErr *MilvusError
			assert.True(t, errors.As(err, &milvusErr))
			assert.Equal(t, tt.op, milvusErr.Op)
			assert.Equal(t, tt.err, milvusErr.Err)
			assert.Equal(t, tt.context, milvusErr.Context)
		})
	}
}

func TestWrapError(t *testing.T) {
	t.Run("wrap non-nil error", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrappedErr := wrapError("TestOp", originalErr)

		require.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), "TestOp")
		assert.Contains(t, wrappedErr.Error(), "original error")

		// Verify it's a MilvusError
		var milvusErr *MilvusError
		assert.True(t, errors.As(wrappedErr, &milvusErr))
		assert.Equal(t, "TestOp", milvusErr.Op)
		assert.Equal(t, originalErr, milvusErr.Err)
		assert.Empty(t, milvusErr.Context)
	})

	t.Run("wrap nil error", func(t *testing.T) {
		wrappedErr := wrapError("TestOp", nil)
		assert.Nil(t, wrappedErr)
	})

	t.Run("wrap predefined error", func(t *testing.T) {
		wrappedErr := wrapError("Query", ErrEmptyData)
		require.Error(t, wrappedErr)
		assert.True(t, errors.Is(wrappedErr, ErrEmptyData))
	})
}

func TestErrorChaining(t *testing.T) {
	t.Run("errors.Is works with MilvusError", func(t *testing.T) {
		originalErr := ErrCollectionNameRequired
		wrappedErr := wrapError("CreateCollection", originalErr)

		assert.True(t, errors.Is(wrappedErr, ErrCollectionNameRequired))
	})

	t.Run("errors.As works with MilvusError", func(t *testing.T) {
		originalErr := errors.New("test error")
		wrappedErr := newError("TestOp", originalErr, "test context")

		var milvusErr *MilvusError
		require.True(t, errors.As(wrappedErr, &milvusErr))
		assert.Equal(t, "TestOp", milvusErr.Op)
		assert.Equal(t, "test context", milvusErr.Context)
	})

	t.Run("nested error wrapping", func(t *testing.T) {
		baseErr := errors.New("base error")
		wrappedOnce := wrapError("FirstOp", baseErr)
		wrappedTwice := wrapError("SecondOp", wrappedOnce)

		assert.Contains(t, wrappedTwice.Error(), "SecondOp")
		assert.True(t, errors.Is(wrappedTwice, baseErr))

		var milvusErr *MilvusError
		assert.True(t, errors.As(wrappedTwice, &milvusErr))
	})
}

func TestMilvusErrorFields(t *testing.T) {
	t.Run("all fields populated", func(t *testing.T) {
		err := &MilvusError{
			Op:      "Insert",
			Err:     errors.New("test"),
			Context: "additional info",
		}

		assert.Equal(t, "Insert", err.Op)
		assert.Equal(t, "test", err.Err.Error())
		assert.Equal(t, "additional info", err.Context)
	})

	t.Run("minimal fields", func(t *testing.T) {
		underlyingErr := errors.New("error")
		err := &MilvusError{
			Op:  "Test",
			Err: underlyingErr,
		}

		assert.Equal(t, "Test", err.Op)
		assert.Equal(t, underlyingErr, err.Err)
		assert.Empty(t, err.Context)
	})
}
