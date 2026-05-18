package backends

import (
	"errors"
	"fmt"
	"strings"
)

type BackendError struct {
	Name string
	Err  error
}

func (e *BackendError) Error() string {
	return fmt.Sprintf("backend %q: %v", e.Name, e.Err)
}

func (e *BackendError) Unwrap() error {
	return e.Err
}

type PartialSaveError struct {
	Succeeded []string
	Failed    []*BackendError
}

func (e *PartialSaveError) Error() string {
	parts := make([]string, 0, len(e.Failed))
	for _, f := range e.Failed {
		parts = append(parts, f.Error())
	}
	return fmt.Sprintf("partial save: succeeded=%v, failures=[%s]", e.Succeeded, strings.Join(parts, "; "))
}

func (e *PartialSaveError) Unwrap() []error {
	errs := make([]error, 0, len(e.Failed))
	for _, f := range e.Failed {
		errs = append(errs, f)
	}
	return errs
}

var _ error = (*BackendError)(nil)
var _ error = (*PartialSaveError)(nil)

func asPartial(err error) (*PartialSaveError, bool) {
	var pe *PartialSaveError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}
