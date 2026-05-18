package backends

import (
	"errors"
	"io/fs"
	"sort"
	"time"
)

type NamedBackend struct {
	Name    string
	Backend ReportStorage
}

type MultiBackend struct {
	backends []NamedBackend
}

func NewMultiBackend(backends []NamedBackend) *MultiBackend {
	return &MultiBackend{backends: backends}
}

func (m *MultiBackend) Save(content string, date time.Time) error {
	var (
		succeeded []string
		failed    []*BackendError
	)
	for _, nb := range m.backends {
		err := nb.Backend.Save(content, date)
		if err == nil {
			succeeded = append(succeeded, nb.Name)
			continue
		}
		if pe, ok := asPartial(err); ok {
			for _, s := range pe.Succeeded {
				succeeded = append(succeeded, nb.Name+"."+s)
			}
			for _, f := range pe.Failed {
				failed = append(failed, &BackendError{Name: nb.Name + "." + f.Name, Err: f.Err})
			}
			continue
		}
		failed = append(failed, &BackendError{Name: nb.Name, Err: err})
	}

	if len(failed) == 0 {
		return nil
	}
	return &PartialSaveError{Succeeded: succeeded, Failed: failed}
}

func (m *MultiBackend) LoadReport(date time.Time) (string, error) {
	var lastErr error
	allNotExist := true
	for _, nb := range m.backends {
		content, err := nb.Backend.LoadReport(date)
		if err == nil {
			return content, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			allNotExist = false
		}
		lastErr = err
	}
	if allNotExist {
		return "", fs.ErrNotExist
	}
	return "", lastErr
}

func (m *MultiBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	var (
		best       time.Time
		found      bool
		lastErr    error
		allMissing = true
	)
	for _, nb := range m.backends {
		t, err := nb.Backend.LoadPreviousReport(date)
		if err == nil {
			if !found || t.After(best) {
				best = t
				found = true
			}
			allMissing = false
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			allMissing = false
			lastErr = err
		}
	}
	if found {
		return best, nil
	}
	if allMissing {
		return time.Time{}, fs.ErrNotExist
	}
	return time.Time{}, lastErr
}

func (m *MultiBackend) ListReports() ([]string, error) {
	seen := make(map[string]struct{})
	var lastErr error
	for _, nb := range m.backends {
		reports, err := nb.Backend.ListReports()
		if err != nil {
			lastErr = err
			continue
		}
		for _, r := range reports {
			seen[r] = struct{}{}
		}
	}
	if len(seen) == 0 && lastErr != nil {
		return nil, lastErr
	}

	merged := make([]string, 0, len(seen))
	for r := range seen {
		merged = append(merged, r)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(merged)))
	return merged, nil
}

func (m *MultiBackend) Close() error {
	errs := make([]error, 0, len(m.backends))
	for _, nb := range m.backends {
		if err := nb.Backend.Close(); err != nil {
			errs = append(errs, &BackendError{Name: nb.Name, Err: err})
		}
	}
	return errors.Join(errs...)
}
