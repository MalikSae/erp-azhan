package shared

import (
	"fmt"
	"os"
)

// Config menyimpan konfigurasi aplikasi yang dimuat dari environment variable.
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	AppPort    string
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
	}
}

// DSN membangun Data Source Name untuk koneksi MySQL.
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
