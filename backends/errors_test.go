package backends

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendErrors(t *testing.T) {
	inner := errors.New("boom")

	tests := map[string]struct {
		err    error
		verify func(*testing.T, error)
	}{
		"PartialSaveError errors.As": {
			err: &PartialSaveError{
				Succeeded: []string{"filesystem"},
				Failed:    []*BackendError{{Name: "git", Err: inner}},
			},
			verify: func(t *testing.T, err error) {
				var pe *PartialSaveError
				require.True(t, errors.As(err, &pe))
				assert.Equal(t, []string{"filesystem"}, pe.Succeeded)
				require.Len(t, pe.Failed, 1)
				assert.Equal(t, "git", pe.Failed[0].Name)
			},
		},
		"BackendError unwrap": {
			err: &BackendError{Name: "x", Err: inner},
			verify: func(t *testing.T, err error) {
				assert.Equal(t, inner, errors.Unwrap(err))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.verify(t, tt.err)
		})
	}
}
