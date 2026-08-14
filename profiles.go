package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
	_ "modernc.org/sqlite"
)

const keychainService = "com.dbaccess.desktop"

type ConnectionProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	Username  string `json:"username"`
	SSLMode   string `json:"sslMode"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ProfileStore struct{ db *sql.DB }

func NewProfileStore() (*ProfileStore, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "DB Access")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "profiles.db"))
	if err != nil {
		return nil, err
	}
	store := &ProfileStore{db: db}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS connection_profiles (id TEXT PRIMARY KEY, name TEXT NOT NULL, host TEXT NOT NULL, port INTEGER NOT NULL, database_name TEXT NOT NULL, username TEXT NOT NULL, ssl_mode TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *ProfileStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *ProfileStore) List() ([]ConnectionProfile, error) {
	rows, err := s.db.Query(`SELECT id, name, host, port, database_name, username, ssl_mode, created_at, updated_at FROM connection_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := make([]ConnectionProfile, 0)
	for rows.Next() {
		var p ConnectionProfile
		if err := rows.Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.Database, &p.Username, &p.SSLMode, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *ProfileStore) Save(profile ConnectionProfile, password string) (ConnectionProfile, error) {
	if profile.ID == "" {
		profile.ID = newID()
	}
	if profile.Port == 0 {
		profile.Port = 5432
	}
	if profile.SSLMode == "" {
		profile.SSLMode = "prefer"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if profile.CreatedAt == "" {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	_, err := s.db.Exec(`INSERT INTO connection_profiles (id,name,host,port,database_name,username,ssl_mode,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,host=excluded.host,port=excluded.port,database_name=excluded.database_name,username=excluded.username,ssl_mode=excluded.ssl_mode,updated_at=excluded.updated_at`, profile.ID, profile.Name, profile.Host, profile.Port, profile.Database, profile.Username, profile.SSLMode, profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		return ConnectionProfile{}, err
	}
	if err := keyring.Set(keychainService, profile.ID, password); err != nil {
		return ConnectionProfile{}, fmt.Errorf("Keychainへの保存に失敗しました: %w", err)
	}
	return profile, nil
}

func (s *ProfileStore) Delete(id string) error {
	if _, err := s.db.Exec(`DELETE FROM connection_profiles WHERE id = ?`, id); err != nil {
		return err
	}
	_ = keyring.Delete(keychainService, id)
	return nil
}

func (s *ProfileStore) LoadConfig(id string) (ConnectionConfiguration, error) {
	var config ConnectionConfiguration
	err := s.db.QueryRow(`SELECT host, port, database_name, username, ssl_mode FROM connection_profiles WHERE id = ?`, id).Scan(&config.Host, &config.Port, &config.Database, &config.Username, &config.SSLMode)
	if err != nil {
		return config, err
	}
	config.Password, err = keyring.Get(keychainService, id)
	return config, err
}

func newID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
func marshalProfiles(value any) (string, error) {
	data, err := json.Marshal(value)
	return string(data), err
}
func profileContext() context.Context { return context.Background() }
