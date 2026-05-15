package backends

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type ReportStorage interface {
	SaveReport(content string, date time.Time) error
	LoadReport(date time.Time) (string, error)
	ListReports() ([]string, error)
	LoadPreviousReport(date time.Time) (string, error)
}

var _ ReportStorage = (*SQLiteBackend)(nil)

type SQLiteBackend struct {
	db *sql.DB
}

func NewSQLiteBackend(path string) (*SQLiteBackend, error) {
	if path == "" {
		path = "~/.local/share/nippo/reports.db"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite: %w", err)
	}

	if err := initSchema(db); err != nil {
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

func (s *SQLiteBackend) SaveReport(content string, date time.Time) error {
	query := `
	INSERT OR REPLACE INTO reports (date, content, updated_at)
	VALUES (?, ?, ?);
	`
	_, err := s.db.Exec(query, date.Format("2006-01-02"), content, time.Now())
	return err
}

func (s *SQLiteBackend) LoadReport(date time.Time) (string, error) {
	var content string
	query := `SELECT content FROM reports WHERE date = ?;`
	err := s.db.QueryRow(query, date.Format("2006-01-02")).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("report not found for %s: %w", date.Format("2006-01-02"), err)
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
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		reports = append(reports, date+".md")
	}
	return reports, rows.Err()
}

func (s *SQLiteBackend) LoadPreviousReport(date time.Time) (string, error) {
	var content string
	query := `SELECT content FROM reports WHERE date < ? ORDER BY date DESC LIMIT 1;`
	err := s.db.QueryRow(query, date.Format("2006-01-02")).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no previous report found: %w", err)
		}
		return "", err
	}
	return content, nil
}

func (s *SQLiteBackend) Close() error {
	return s.db.Close()
}
