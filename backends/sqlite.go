package backends

import (
	"database/sql"
	"fmt"
	"io/fs"
	"time"

	"github.com/adrg/xdg"
	_ "modernc.org/sqlite"
)

type SQLiteBackend struct {
	db *sql.DB
}

var _ ReportStorage = (*SQLiteBackend)(nil)

func NewSQLiteBackend(path string) (*SQLiteBackend, error) {
	if path == "" {
		var err error
		path, err = xdg.DataFile("nippo/reports.db")
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite: %w", err)
	}

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return &SQLiteBackend{db: db}, nil
}

func initSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS reports (
		date TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	return err
}

func (s *SQLiteBackend) Save(content string, date time.Time) error {
	normalized := normalizeReportDate(date)
	query := `
	INSERT INTO reports (date, content, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(date) DO UPDATE SET
		content = excluded.content,
		updated_at = CURRENT_TIMESTAMP;
	`
	_, err := s.db.Exec(query, normalized.Format("2006-01-02"), content)
	return err
}

func (s *SQLiteBackend) LoadReport(date time.Time) (string, error) {
	var content string
	normalized := normalizeReportDate(date)
	query := `SELECT content FROM reports WHERE date = ?;`
	err := s.db.QueryRow(query, normalized.Format("2006-01-02")).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("report not found for %s: %w", normalized.Format("2006-01-02"), fs.ErrNotExist)
		}
		return "", err
	}
	return content, nil
}

func (s *SQLiteBackend) ListReports() ([]string, error) {
	query := `SELECT date FROM reports ORDER BY date DESC;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []string
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return nil, err
		}
		t, err := parseSQLiteDate(dateStr)
		if err != nil {
			return nil, err
		}
		reports = append(reports, t.Format("2006/01/02")+".md")
	}
	return reports, rows.Err()
}

func (s *SQLiteBackend) LoadPreviousReport(date time.Time) (time.Time, error) {
	var dateStr string
	normalized := normalizeReportDate(date)
	query := `SELECT date FROM reports WHERE date < ? ORDER BY date DESC LIMIT 1;`
	err := s.db.QueryRow(query, normalized.Format("2006-01-02")).Scan(&dateStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, fs.ErrNotExist
		}
		return time.Time{}, err
	}
	return parseSQLiteDate(dateStr)
}

func (s *SQLiteBackend) Close() error {
	return s.db.Close()
}

func parseSQLiteDate(date string) (time.Time, error) {
	return time.Parse("2006-01-02", date)
}
