package backends

import (
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiBackend_SaveAllSuccess(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, m.Save("x", date))

	assert.Equal(t, 1, a.saveCalls)
	assert.Equal(t, 1, b.saveCalls)
}

func TestMultiBackend_SavePartial(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	b.saveErr = errMockBoom
	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	err := m.Save("x", date)
	require.Error(t, err)

	var pe *PartialSaveError
	require.True(t, errors.As(err, &pe))
	assert.Equal(t, []string{"a"}, pe.Succeeded)
	require.Len(t, pe.Failed, 1)
	assert.Equal(t, "b", pe.Failed[0].Name)
}

func TestMultiBackend_LoadReportFallback(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, b.Save("from-b", date))

	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
	got, err := m.LoadReport(date)
	require.NoError(t, err)
	assert.Equal(t, "from-b", got)
}

func TestMultiBackend_LoadReportAllNotExist(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})

	_, err := m.LoadReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestMultiBackend_ListReportsDedup(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, a.Save("x", date))
	require.NoError(t, b.Save("y", date))

	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
	got, err := m.ListReports()
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestMultiBackend_CloseJoinsErrors(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	a.closeErr = errMockBoom
	b.closeErr = errMockBoom

	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
	err := m.Close()
	require.Error(t, err)
	assert.True(t, a.closed)
	assert.True(t, b.closed)
}

func TestMultiBackend_LoadPreviousReportPicksLatest(t *testing.T) {
	a := newMockBackend()
	b := newMockBackend()
	require.NoError(t, a.Save("a-old", time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, b.Save("b-newer", time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)))

	m := NewMultiBackend([]NamedBackend{{Name: "a", Backend: a}, {Name: "b", Backend: b}})
	got, err := m.LoadPreviousReport(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC), got)
}
