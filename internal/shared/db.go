package shared

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// NewDB membuka koneksi ke MySQL menggunakan DSN dari Config.
// Mengembalikan *sql.DB yang sudah diverifikasi Ping-nya.
func NewDB(cfg *Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("db: gagal membuka koneksi: %w", err)
	}

	// Konfigurasi connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Verifikasi koneksi aktif
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("db: gagal ping ke database '%s': %w", cfg.DBName, err)
	}

	return db, nil
}
