package backends

import (
	"errors"
	"io/fs"
	"sort"
	"sync"
	"time"

	"github.com/nurazon59/nippo/report"
)

// mockBackend は backends パッケージ内テスト専用のスタブ。
// legacy (Save/LoadReport) と v1 (SaveReport/LoadReportStruct/WriteSidecar) を
// 同じ instance で観測したいので、保存先 map を 3 つに分けている。
type mockBackend struct {
	mu              sync.Mutex
	data            map[string]string
	structData      map[string]*report.Report
	sidecars        map[string]map[string][]byte
	saveErr         error
	loadErr         error
	listErr         error
	saveReportErr   error
	loadStructErr   error
	sidecarErr      error
	closeErr        error
	closed          bool
	saveCalls       int
	saveReportCalls int
	sidecarCalls    int
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		data:       make(map[string]string),
		structData: make(map[string]*report.Report),
		sidecars:   make(map[string]map[string][]byte),
	}
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

func (m *mockBackend) LoadLatestReport() (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return time.Time{}, m.loadErr
	}
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
		return t, nil
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

func (m *mockBackend) SaveReport(r *report.Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveReportCalls++
	if m.saveReportErr != nil {
		return m.saveReportErr
	}
	// deep copy までは要らないが、外部からの mutation を防ぐため value copy で保持する。
	cp := *r
	m.structData[normalizeReportDate(r.Date).Format("2006-01-02")] = &cp
	return nil
}

func (m *mockBackend) LoadReportStruct(date time.Time) (*report.Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadStructErr != nil {
		return nil, m.loadStructErr
	}
	v, ok := m.structData[normalizeReportDate(date).Format("2006-01-02")]
	if !ok {
		return nil, fs.ErrNotExist
	}
	cp := *v
	return &cp, nil
}

func (m *mockBackend) WriteSidecar(date time.Time, kind string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sidecarCalls++
	if m.sidecarErr != nil {
		return m.sidecarErr
	}
	key := normalizeReportDate(date).Format("2006-01-02")
	bucket, ok := m.sidecars[key]
	if !ok {
		bucket = make(map[string][]byte)
		m.sidecars[key] = bucket
	}
	// 呼び出し側が再利用するスライスを後から書き換える可能性を排除するため copy する。
	buf := make([]byte, len(content))
	copy(buf, content)
	bucket[kind] = buf
	return nil
}

var errMockBoom = errors.New("mock boom")
