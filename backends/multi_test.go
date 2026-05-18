package backends

import (
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiBackend_Save(t *testing.T) {
	tests := map[string]struct {
		setupB        func(b *mockBackend)
		wantSucceeded []string
		wantFailed    []string
	}{
		"all success": {
			setupB:        func(b *mockBackend) {},
			wantSucceeded: nil,
			wantFailed:    nil,
		},
		"partial failure": {
			setupB:        func(b *mockBackend) { b.saveErr = errMockBoom },
			wantSucceeded: []string{"a"},
			wantFailed:    []string{"b"},
		},
	}

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := newMockBackend()
			b := newMockBackend()
			tt.setupB(b)

			m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
			err := m.Save("x", date)

			if tt.wantFailed == nil {
				require.NoError(t, err)
				return
			}
			var pe *PartialSaveError
			require.True(t, errors.As(err, &pe))
			assert.Equal(t, tt.wantSucceeded, pe.Succeeded)
			gotFailed := make([]string, 0, len(pe.Failed))
			for _, f := range pe.Failed {
				gotFailed = append(gotFailed, f.Name)
			}
			assert.Equal(t, tt.wantFailed, gotFailed)
		})
	}
}

func TestMultiBackend_LoadReport(t *testing.T) {
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setupA  func(b *mockBackend)
		setupB  func(b *mockBackend)
		wantErr bool
		wantIs  error
		want    string
	}{
		"fallback to second on miss in first": {
			setupA: func(b *mockBackend) {},
			setupB: func(b *mockBackend) { require.NoError(t, b.Save("from-b", date)) },
			want:   "from-b",
		},
		"first hit wins": {
			setupA: func(b *mockBackend) { require.NoError(t, b.Save("from-a", date)) },
			setupB: func(b *mockBackend) { require.NoError(t, b.Save("from-b", date)) },
			want:   "from-a",
		},
		"all not exist returns fs.ErrNotExist": {
			setupA:  func(b *mockBackend) {},
			setupB:  func(b *mockBackend) {},
			wantErr: true,
			wantIs:  fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := newMockBackend()
			b := newMockBackend()
			tt.setupA(a)
			tt.setupB(b)

			m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
			got, err := m.LoadReport(date)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantIs != nil {
					assert.True(t, errors.Is(err, tt.wantIs))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMultiBackend_LoadPreviousReport(t *testing.T) {
	target := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		datesA  []string
		datesB  []string
		want    string
		wantErr bool
	}{
		"picks latest across backends": {
			datesA: []string{"2024-06-10"},
			datesB: []string{"2024-06-14"},
			want:   "2024-06-14",
		},
		"empty everywhere errors": {
			datesA:  nil,
			datesB:  nil,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := newMockBackend()
			b := newMockBackend()
			for _, d := range tt.datesA {
				require.NoError(t, a.Save("x", mustDate(t, d)))
			}
			for _, d := range tt.datesB {
				require.NoError(t, b.Save("x", mustDate(t, d)))
			}

			m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
			got, err := m.LoadPreviousReport(target)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, mustDate(t, tt.want), got)
		})
	}
}

func TestMultiBackend_ListReports(t *testing.T) {
	tests := map[string]struct {
		datesA  []string
		datesB  []string
		wantLen int
	}{
		"dedupes overlapping entries": {
			datesA:  []string{"2024-06-15"},
			datesB:  []string{"2024-06-15"},
			wantLen: 1,
		},
		"merges distinct entries": {
			datesA:  []string{"2024-06-15"},
			datesB:  []string{"2024-06-14"},
			wantLen: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := newMockBackend()
			b := newMockBackend()
			for _, d := range tt.datesA {
				require.NoError(t, a.Save("x", mustDate(t, d)))
			}
			for _, d := range tt.datesB {
				require.NoError(t, b.Save("x", mustDate(t, d)))
			}

			m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
			got, err := m.ListReports()
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestMultiBackend_Close(t *testing.T) {
	tests := map[string]struct {
		setupA  func(b *mockBackend)
		setupB  func(b *mockBackend)
		wantErr bool
	}{
		"both succeed": {
			setupA: func(b *mockBackend) {},
			setupB: func(b *mockBackend) {},
		},
		"errors are joined": {
			setupA:  func(b *mockBackend) { b.closeErr = errMockBoom },
			setupB:  func(b *mockBackend) { b.closeErr = errMockBoom },
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := newMockBackend()
			b := newMockBackend()
			tt.setupA(a)
			tt.setupB(b)

			m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
			err := m.Close()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.True(t, a.closed)
			assert.True(t, b.closed)
		})
	}
}
