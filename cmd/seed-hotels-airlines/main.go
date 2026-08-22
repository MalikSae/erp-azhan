package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type hotelSeed struct {
	Name       string
	City       string
	StarRating int
	DistanceM  int
}

// Nama hotel nyata. DistanceM adalah estimasi jarak jalan kaki operasional ke
// Masjidil Haram/Masjid Nabawi dan tetap dapat dikoreksi admin setelah seeding.
var hotels = []hotelSeed{
	{"Fairmont Makkah Clock Royal Tower", "mekkah", 5, 100},
	{"Swissotel Makkah", "mekkah", 5, 150},
	{"Pullman ZamZam Makkah", "mekkah", 5, 150},
	{"Makkah Towers", "mekkah", 5, 200},
	{"Jabal Omar Marriott Hotel Makkah", "mekkah", 5, 650},
	{"DoubleTree by Hilton Makkah Jabal Omar", "mekkah", 4, 750},
	{"Anjum Hotel Makkah", "mekkah", 5, 750},
	{"voco Makkah by IHG", "mekkah", 5, 1400},
	{"Anwar Al Madinah Mövenpick", "madinah", 5, 100},
	{"Dallah Taibah Hotel", "madinah", 5, 100},
	{"Pullman Zamzam Madina", "madinah", 5, 150},
	{"Frontel Al Harithia", "madinah", 5, 200},
	{"Millennium Al Aqeeq Hotel", "madinah", 5, 250},
	{"Saja Al Madinah", "madinah", 4, 600},
	{"Leader Al Muna Kareem Hotel", "madinah", 4, 500},
	{"Elaf Taiba Hotel", "madinah", 3, 350},
}

var airlines = []string{
	"Garuda Indonesia",
	"Saudia",
	"Emirates",
	"Qatar Airways",
	"Etihad Airways",
	"Oman Air",
	"Turkish Airlines",
	"Malaysia Airlines",
	"Royal Brunei Airlines",
	"Lion Air",
	"Batik Air",
	"AirAsia X",
}

func main() {
	_ = godotenv.Load()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		env("DB_USER", "root"), env("DB_PASSWORD", ""), env("DB_HOST", "127.0.0.1"),
		env("DB_PORT", "3306"), env("DB_NAME", "erp_azhan_dev"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("[ERROR] membuka database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("[ERROR] koneksi database: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("[ERROR] memulai transaksi: %v", err)
	}
	defer tx.Rollback()

	hotelCreated, hotelSkipped := seedHotels(ctx, tx)
	airlineCreated, airlineSkipped := seedAirlines(ctx, tx)
	if err := tx.Commit(); err != nil {
		log.Fatalf("[ERROR] commit seeder: %v", err)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SEED MASTER HOTEL & MASKAPAI SELESAI")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Hotel    : %d dibuat, %d sudah ada/dilewati\n", hotelCreated, hotelSkipped)
	fmt.Printf("Maskapai : %d dibuat, %d sudah ada/dilewati\n", airlineCreated, airlineSkipped)
	fmt.Println("Foto dan logo sengaja kosong; unggah aset resmi melalui dashboard.")
}

func seedHotels(ctx context.Context, tx *sql.Tx) (created, skipped int) {
	for _, item := range hotels {
		var id uint64
		err := tx.QueryRowContext(ctx, `SELECT id FROM hotels WHERE UPPER(name)=UPPER(?) LIMIT 1`, item.Name).Scan(&id)
		if err == nil {
			skipped++
			continue
		}
		if err != sql.ErrNoRows {
			log.Fatalf("[ERROR] memeriksa hotel %q: %v", item.Name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO hotels (name, city, star_rating, distance_m, photo_url) VALUES (?, ?, ?, ?, NULL)`, item.Name, item.City, item.StarRating, item.DistanceM); err != nil {
			log.Fatalf("[ERROR] menambah hotel %q: %v", item.Name, err)
		}
		created++
	}
	return
}

func seedAirlines(ctx context.Context, tx *sql.Tx) (created, skipped int) {
	for _, name := range airlines {
		var id uint64
		err := tx.QueryRowContext(ctx, `SELECT id FROM airlines WHERE UPPER(name)=UPPER(?) LIMIT 1`, name).Scan(&id)
		if err == nil {
			skipped++
			continue
		}
		if err != sql.ErrNoRows {
			log.Fatalf("[ERROR] memeriksa maskapai %q: %v", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO airlines (name, logo_url) VALUES (?, NULL)`, name); err != nil {
			log.Fatalf("[ERROR] menambah maskapai %q: %v", name, err)
		}
		created++
	}
	return
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
