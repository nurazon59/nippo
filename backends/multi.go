package backends

import (
	"errors"
	"io/fs"
	"sort"
	"time"
)

type MultiBackend struct {
	backends []ReportStorage
}

func NewMultiBackend(backends []ReportStorage) *MultiBackend {
	return &MultiBackend{backends: backends}
}

func (m *MultiBackend) Save(content string, date time.Time) error {
	var errs []error
	for _, b := range m.backends {
		if err := b.Save(content, date); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *MultiBackend) LoadReport(date time.Time) (string, error) {
	var lastErr error
	for _, b := range m.backends {
		content, err := b.LoadReport(date)
		if err == nil {
			return content, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func (m *MultiBackend) ListReports() ([]string, error) {
	seen := make(map[string]bool)
	var reports []string

	for _, b := range m.backends {
		list, err := b.ListReports()
		if err != nil {
			continue
		}
		for _, r := range list {
			if !seen[r] {
				seen[r] = true
				reports = append(reports, r)
			}
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(reports)))
	return reports, nil
}

func (m *MultiBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	var candidates []time.Time
	var lastErr error

	for _, b := range m.backends {
		previousDate, err := b.LoadPreviousReport(date)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				lastErr = err
			}
			continue
		}
		candidates = append(candidates, previousDate)
	}

	if len(candidates) == 0 {
		if lastErr != nil {
			return time.Time{}, lastErr
		}
		return time.Time{}, fs.ErrNotExist
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].After(candidates[j])
	})

	return candidates[0], nil
}
