package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run cmd/seed-admin/main.go <email> <password>")
		os.Exit(1)
	}

	email := os.Args[1]
	password := os.Args[2]

	if !strings.Contains(email, "@") {
		fmt.Println("[ERROR] Email tidak valid (harus mengandung '@')")
		os.Exit(1)
	}

	if len(password) < 8 {
		fmt.Println("[ERROR] Password minimal 8 karakter")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("[ERROR] Gagal hash password: %v\n", err)
		os.Exit(1)
	}

	// Load config
	if err := godotenv.Load(); err != nil {
		// Abaikan, pakai system env
	}

	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "erp_azhan_dev")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("[ERROR] Gagal koneksi DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO admin_users (email, password_hash) VALUES (?, ?)", email, string(hash))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errorsAs(err, &mysqlErr) && mysqlErr.Number == 1062 { // Duplicate entry
			fmt.Println("[ERROR] email sudah terdaftar")
			os.Exit(1)
		}
		fmt.Printf("[ERROR] Gagal insert admin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[OK] Admin berhasil dibuat: %s\n", email)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// errorsAs replaces errors.As to avoid importing "errors" just for this check
func errorsAs(err error, target any) bool {
	// Simple type assertion is not perfect but works for this specific case where err is often mysql.MySQLError pointer
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if t, ok := target.(**mysql.MySQLError); ok {
			*t = mysqlErr
			return true
		}
	}
	return false
}
