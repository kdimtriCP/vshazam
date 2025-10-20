package database

import (
	"database/sql"
	"fmt"

	"github.com/kdimtricp/vshazam/internal/models/frame_analysis"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/kdimtricp/vshazam/internal/models"
)

type DB struct {
	gormDB *gorm.DB
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
	var gormDB *gorm.DB
	var err error

	switch config.Type {
	case "sqlite":
		gormDB, err = gorm.Open(sqlite.Open(config.SQLitePath), &gorm.Config{})
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.Host, config.Port, config.User, config.Password, config.Name)
		gormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{gormDB: gormDB, conn: sqlDB, dbType: config.Type}

	if err := gormDB.AutoMigrate(&models.Video{}, &frame_analysis.FrameAnalysisDB{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return db, nil
}

// RunMigrations executes all pending migrations
func (db *DB) RunMigrations(migrationsPath string) error {
	migrator := NewMigrator(db.conn, db.dbType)
	return migrator.Run(migrationsPath)
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) GORM() *gorm.DB {
	return db.gormDB
}
