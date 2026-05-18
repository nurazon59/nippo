package backends

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialSaveError_ErrorsAs(t *testing.T) {
	inner := errors.New("disk full")
	pe := &PartialSaveError{
		Succeeded: []string{"filesystem"},
		Failed:    []*BackendError{{Name: "git", Err: inner}},
	}

	var got *PartialSaveError
	require.True(t, errors.As(error(pe), &got))
	assert.Equal(t, []string{"filesystem"}, got.Succeeded)
}

func TestBackendError_Unwrap(t *testing.T) {
	inner := errors.New("boom")
	be := &BackendError{Name: "x", Err: inner}
	assert.Equal(t, inner, errors.Unwrap(be))
}
