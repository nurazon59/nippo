package backends

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteBackend struct {
	db   *sql.DB
	path string
}

func NewSQLiteBackend(path string) (*SQLiteBackend, error) {
	if path == "" {
		return nil, errors.New("sqlite backend: path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("sqlite backend: mkdir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000", url.QueryEscape(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite backend: open: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reports (
			date TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite backend: create table: %w", err)
	}

	return &SQLiteBackend{db: db, path: path}, nil
}

func sqliteDateKey(date time.Time) string {
	return normalizeReportDate(date).Format("2006-01-02")
}

func (b *SQLiteBackend) Save(content string, date time.Time) error {
	key := sqliteDateKey(date)
	_, err := b.db.Exec(
		`INSERT INTO reports(date, content, updated_at) VALUES(?, ?, ?)
		 ON CONFLICT(date) DO UPDATE SET content = excluded.content, updated_at = excluded.updated_at`,
		key, content, time.Now().Unix(),
	)
	return err
}

func (b *SQLiteBackend) LoadReport(date time.Time) (string, error) {
	key := sqliteDateKey(date)
	var content string
	err := b.db.QueryRow(`SELECT content FROM reports WHERE date = ?`, key).Scan(&content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fs.ErrNotExist
		}
		return "", err
	}
	return content, nil
}

func (b *SQLiteBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	target := sqliteDateKey(date)
	var key string
	err := b.db.QueryRow(
		`SELECT date FROM reports WHERE date < ? ORDER BY date DESC LIMIT 1`, target,
	).Scan(&key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, fs.ErrNotExist
		}
		return time.Time{}, err
	}
	return time.Parse("2006-01-02", key)
}

func (b *SQLiteBackend) ListReports() ([]string, error) {
	rows, err := b.db.Query(`SELECT date FROM reports ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		t, err := time.Parse("2006-01-02", key)
		if err != nil {
			continue
		}
		reports = append(reports, filepath.Join(t.Format("2006"), t.Format("01"), t.Format("02")+".md"))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reports, nil
}

func (b *SQLiteBackend) Close() error {
	return b.db.Close()
}
