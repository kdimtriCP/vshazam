package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn   *sql.DB
	dbType string
}

type Config struct {
	Type       string
	Host       string
	Port       int
	User       string
	Password   string
	Name       string
	SQLitePath string
}

func NewDB(config Config) (*DB, error) {
	var conn *sql.DB
	var err error

	switch config.Type {
	case "sqlite":
		conn, err = sql.Open("sqlite3", config.SQLitePath)
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.Host, config.Port, config.User, config.Password, config.Name)
		conn, err = sql.Open("pgx", dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn, dbType: config.Type}

	// Only create tables for SQLite
	if config.Type == "sqlite" {
		if err := db.createTables(); err != nil {
			return nil, fmt.Errorf("failed to create tables: %w", err)
		}
	}

	return db, nil
}

func (db *DB) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS videos (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		upload_time DATETIME NOT NULL
	);
	`

	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}
