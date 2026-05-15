package backends

import (
	"errors"
	"fmt"
	"io/fs"
	"time"
)

type MultiBackend struct {
	backends []ReportStorage
}

func NewMultiBackend(backends []ReportStorage) *MultiBackend {
	return &MultiBackend{backends: backends}
}

func (m *MultiBackend) SaveReport(content string, date time.Time) error {
	var errs []error
	for _, b := range m.backends {
		if err := b.SaveReport(content, date); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to save to some backends: %v", errs)
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

	return reports, nil
}

func (m *MultiBackend) LoadPreviousReport(date time.Time) (string, error) {
	var lastErr error
	for _, b := range m.backends {
		content, err := b.LoadPreviousReport(date)
		if err == nil {
			return content, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			lastErr = err
		}
	}
	return "", lastErr
}
