package shared

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

// Config menyimpan konfigurasi aplikasi yang dimuat dari environment variable.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	AppPort    string
	// Timezone adalah zona waktu bisnis (default Asia/Jakarta). Seluruh kolom
	// DATETIME, NOW(), dan CURRENT_DATE memakai zona ini, baik dari MySQL
	// maupun dari driver Go, supaya waktu yang ditulis dan dibaca konsisten.
	Timezone string
}

// LoadConfig memuat konfigurasi dari environment variable.
// Pastikan godotenv.Load() sudah dipanggil sebelum fungsi ini.
func LoadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "erp_azhan_dev"),
		AppPort:    getEnv("APP_PORT", "8080"),
		Timezone:   getEnv("APP_TIMEZONE", "Asia/Jakarta"),
	}
}

// Location mengembalikan zona waktu bisnis. Nama zona yang tidak dikenal
// jatuh ke Asia/Jakarta (binary menyertakan time/tzdata).
func (c *Config) Location() *time.Location {
	if loc, err := time.LoadLocation(c.Timezone); err == nil {
		return loc
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return loc
}

// DSN membangun Data Source Name untuk koneksi MySQL. loc membuat driver
// membaca/menulis DATETIME dalam zona bisnis; time_zone membuat NOW() dan
// CURRENT_DATE di MySQL memakai zona yang sama, tidak bergantung pada
// zona sistem server database.
func (c *Config) DSN() string {
	loc := c.Location()
	offset := time.Now().In(loc).Format("-07:00")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=%s&time_zone=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
		url.QueryEscape(loc.String()), url.QueryEscape("'"+offset+"'"))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
