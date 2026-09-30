package testdb

import (
	"database/sql"
	"erp-azhan/api/internal/shared"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Portal opens a dedicated one-connection pool. Temporary tables shadow real
// tables only on this connection; closing it removes every synthetic row.
// No persistent migration or real customer data is changed.
func Portal(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("ERP_TEST_DB") != "1" {
		t.Skip("set ERP_TEST_DB=1 for isolated MySQL tests")
	}
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	db, err := shared.NewDB(shared.LoadConfig())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { db.Close() })
	statements := []string{
		`CREATE TEMPORARY TABLE jamaah(id BIGINT UNSIGNED PRIMARY KEY,brand_id BIGINT UNSIGNED NOT NULL,portal_pin_hash VARCHAR(100))`,
		`CREATE TEMPORARY TABLE dokumen_jamaah(id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,jamaah_id BIGINT UNSIGNED NOT NULL,jenis VARCHAR(40) NOT NULL,file_url VARCHAR(500),status VARCHAR(30),updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,UNIQUE KEY(jamaah_id,jenis))`,
	}
	for _, stmt := range statements {
		if _, err = db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "migrations", "066_portal_security_documents.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var clean []string
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			clean = append(clean, line)
		}
	}
	for _, stmt := range strings.Split(strings.Join(clean, "\n"), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		stmt = strings.Replace(stmt, "CREATE TABLE IF NOT EXISTS", "CREATE TEMPORARY TABLE", 1)
		if _, err = db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
