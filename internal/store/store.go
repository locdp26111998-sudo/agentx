package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// SessionStore lưu và lấy session_id theo sender_id.
type SessionStore interface {
	GetSession(senderID string) (sessionID string, err error)
	SaveSession(senderID, sessionID string) error
}

// SQLiteStore implement SessionStore dùng SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore mở/tạo DB, bảng sessions và bật WAL.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("tạo thư mục DB: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("mở sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("bật WAL: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			sender_id  TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("tạo bảng sessions: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) GetSession(senderID string) (string, error) {
	var sessionID string
	err := s.db.QueryRow(
		`SELECT session_id FROM sessions WHERE sender_id = ?`, senderID,
	).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (s *SQLiteStore) SaveSession(senderID, sessionID string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sessions (sender_id, session_id, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, senderID, sessionID)
	return err
}

// SessionCount trả số phiên hội thoại đang lưu (dùng admin hiển thị online/offline).
func (s *SQLiteStore) SessionCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	return n, err
}
