//go:build ignore

// Script untuk menjalankan file migrasi SQL ke database erp_azhan_dev.
// Pastikan file .env sudah diisi sebelum menjalankan.
//
// Cara menjalankan:
//   go run migrations/run.go
//
// Syarat:
//   - Database erp_azhan_dev sudah dibuat secara manual (via HeidiSQL / phpMyAdmin Laragon)
//   - File .env sudah diisi dengan kredensial MySQL

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] File .env tidak ditemukan, mencoba dari environment system...")
	}

	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "erp_azhan_dev")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("[ERROR] Gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("[ERROR] Tidak dapat terhubung ke database '%s': %v", dbName, err)
	}

	log.Printf("[INFO] Terhubung ke database: %s", dbName)

	// Baca file SQL — default 001_init.sql, bisa di-override via argumen CLI
	// Contoh: go run migrations/run.go migrations/002_schedules_fix.sql
	sqlFile := "migrations/001_init.sql"
	if len(os.Args) > 1 {
		sqlFile = os.Args[1]
	}
	content, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("[ERROR] Gagal membaca file %s: %v", sqlFile, err)
	}

	// Pisahkan dan jalankan per statement (hapus komentar --... dan baris kosong)
	stmts := splitStatements(string(content))
	for i, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		log.Printf("[RUN] Statement %d: %.60s...", i+1, stmt)
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("[ERROR] Gagal menjalankan statement %d: %v\nSQL: %s", i+1, err, stmt)
		}
	}

	log.Println("[OK] Migrasi berhasil. Semua tabel telah dibuat.")
}

// splitStatements memisahkan SQL berdasarkan semicolon, mengabaikan baris komentar.
func splitStatements(sql string) []string {
	var result []string
	var current strings.Builder

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip komentar baris tunggal
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")

		// Jika baris berakhiran ;, itu adalah akhir statement
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				result = append(result, stmt)
			}
			current.Reset()
		}
	}

	return result
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
