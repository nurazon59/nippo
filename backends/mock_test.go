package backends

import (
	"errors"
	"io/fs"
	"sort"
	"sync"
	"time"
)

type mockBackend struct {
	mu        sync.Mutex
	data      map[string]string
	saveErr   error
	loadErr   error
	listErr   error
	closeErr  error
	closed    bool
	saveCalls int
}

func newMockBackend() *mockBackend {
	return &mockBackend{data: make(map[string]string)}
}

func (m *mockBackend) Save(content string, date time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCalls++
	if m.saveErr != nil {
		return m.saveErr
	}
	m.data[normalizeReportDate(date).Format("2006-01-02")] = content
	return nil
}

func (m *mockBackend) LoadReport(date time.Time) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return "", m.loadErr
	}
	v, ok := m.data[normalizeReportDate(date).Format("2006-01-02")]
	if !ok {
		return "", fs.ErrNotExist
	}
	return v, nil
}

func (m *mockBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return time.Time{}, m.loadErr
	}
	target := normalizeReportDate(date)
	var keys []string
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	for _, k := range keys {
		t, err := time.Parse("2006-01-02", k)
		if err != nil {
			continue
		}
		if t.Before(target) {
			return t, nil
		}
	}
	return time.Time{}, fs.ErrNotExist
}

func (m *mockBackend) ListReports() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []string
	for k := range m.data {
		t, _ := time.Parse("2006-01-02", k)
		out = append(out, t.Format("2006/01/02")+".md")
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out, nil
}

func (m *mockBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeErr
}

var errMockBoom = errors.New("mock boom")
