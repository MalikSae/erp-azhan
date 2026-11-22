package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type hotelSeed struct {
	Name       string
	City       string
	StarRating int
	DistanceM  int
	LogoURL    string
}

type airlineSeed struct {
	Name    string
	Code    string
	LogoURL string
}

// Nama hotel nyata. DistanceM adalah estimasi jarak jalan kaki operasional ke
// Masjidil Haram/Masjid Nabawi dan tetap dapat dikoreksi admin setelah seeding.
var hotels = []hotelSeed{
	{"Fairmont Makkah Clock Royal Tower", "Makkah", 5, 100, "/uploads/hotel-logos/fairmont.png"},
	{"Swissotel Makkah", "Makkah", 5, 150, ""},
	{"Pullman ZamZam Makkah", "Makkah", 5, 150, ""},
	{"Makkah Towers", "Makkah", 5, 200, ""},
	{"Jabal Omar Marriott Hotel Makkah", "Makkah", 5, 650, ""},
	{"DoubleTree by Hilton Makkah Jabal Omar", "Makkah", 4, 750, "/uploads/hotel-logos/doubletree.png"},
	{"Anjum Hotel Makkah", "Makkah", 5, 750, "/uploads/hotel-logos/anjum.png"},
	{"voco Makkah by IHG", "Makkah", 5, 1400, "/uploads/hotel-logos/voco.png"},
	{"Anwar Al Madinah Mövenpick", "Madinah", 5, 100, "/uploads/hotel-logos/movenpick.png"},
	{"Dallah Taibah Hotel", "Madinah", 5, 100, "/uploads/hotel-logos/dallah-taibah.png"},
	{"Pullman Zamzam Madina", "Madinah", 5, 150, ""},
	{"Frontel Al Harithia", "Madinah", 5, 200, ""},
	{"Millennium Al Aqeeq Hotel", "Madinah", 5, 250, ""},
	{"Saja Al Madinah", "Madinah", 4, 600, ""},
	{"Leader Al Muna Kareem Hotel", "Madinah", 4, 500, ""},
	{"Elaf Taiba Hotel", "Madinah", 3, 350, ""},
}

var airlines = []airlineSeed{
	{"Garuda Indonesia", "GA", "/uploads/airline-logos/garuda-indonesia.webp"},
	{"Saudia", "SV", "/uploads/airline-logos/saudia.jpeg"},
	{"Emirates", "EK", "/uploads/airline-logos/emirates.webp"},
	{"Etihad Airways", "EY", "/uploads/airline-logos/etihad-airways.webp"},
	{"Qatar Airways", "QR", "/uploads/airline-logos/qatar-airways.png"},
	{"Oman Air", "WY", "/uploads/airline-logos/oman-air.webp"},
	{"Turkish Airlines", "TK", "/uploads/airline-logos/turkish-airlines.webp"},
	{"Flynas", "XY", ""},
	{"Malaysia Airlines", "MH", "/uploads/airline-logos/malaysia-airlines.png"},
	{"Singapore Airlines", "SQ", ""},
	{"Scoot", "TR", ""},
	{"Royal Brunei Airlines", "BI", ""},
	{"Lion Air", "JT", "/uploads/airline-logos/lion-air.webp"},
	{"Batik Air", "ID", ""},
	{"Citilink", "QG", "/uploads/airline-logos/citilink.png"},
	{"Pelita Air", "IP", ""},
	{"AirAsia X", "D7", ""},
	{"Kuwait Airways", "KU", ""},
	{"Gulf Air", "GF", ""},
	{"EgyptAir", "MS", "/uploads/airline-logos/egyptair.webp"},
	// Tambahan (logo dari folder IT/maskapai, 2026-11-22). AirAsia (AK) terpisah dari AirAsia X (D7): logo AirAsia X belum ada.
	{"Air Arabia", "G9", "/uploads/airline-logos/air-arabia.png"},
	{"Flyadeal", "F3", "/uploads/airline-logos/flyadeal.png"},
	{"Ethiopian Airlines", "ET", "/uploads/airline-logos/ethiopian-airlines.png"},
	{"Hainan Airlines", "HU", "/uploads/airline-logos/hainan-airlines.png"},
	{"IndiGo", "6E", "/uploads/airline-logos/indigo.webp"},
	{"AirAsia", "AK", "/uploads/airline-logos/airasia.webp"},
	{"Uzbekistan Airways", "HY", "/uploads/airline-logos/uzbekistan-airways.webp"},
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

	hotelCreated, hotelUpdated, hotelSkipped := seedHotels(ctx, tx)
	airlineCreated, airlineUpdated, airlineSkipped := seedAirlines(ctx, tx)
	if err := tx.Commit(); err != nil {
		log.Fatalf("[ERROR] commit seeder: %v", err)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SEED MASTER HOTEL & MASKAPAI SELESAI")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Hotel    : %d dibuat, %d logo dilengkapi, %d sudah sesuai/dilewati\n", hotelCreated, hotelUpdated, hotelSkipped)
	fmt.Printf("Maskapai : %d dibuat, %d diperbarui (kode/logo), %d sudah sesuai/dilewati\n", airlineCreated, airlineUpdated, airlineSkipped)
	fmt.Println("Logo yang belum tersedia dapat diunggah melalui dashboard.")
}

func seedHotels(ctx context.Context, tx *sql.Tx) (created, updated, skipped int) {
	for _, item := range hotels {
		var id uint64
		var photoURL sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT id, photo_url FROM hotels WHERE UPPER(name)=UPPER(?) LIMIT 1`, item.Name).Scan(&id, &photoURL)
		if err == nil {
			if item.LogoURL != "" && (!photoURL.Valid || photoURL.String == "") {
				if _, err := tx.ExecContext(ctx, `UPDATE hotels SET photo_url=? WHERE id=?`, item.LogoURL, id); err != nil {
					log.Fatalf("[ERROR] memperbarui logo hotel %q: %v", item.Name, err)
				}
				updated++
				continue
			}
			skipped++
			continue
		}
		if err != sql.ErrNoRows {
			log.Fatalf("[ERROR] memeriksa hotel %q: %v", item.Name, err)
		}
		var seedLogo any
		if item.LogoURL != "" {
			seedLogo = item.LogoURL
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO hotels (name, city, star_rating, distance_m, photo_url) VALUES (?, ?, ?, ?, ?)`, item.Name, item.City, item.StarRating, item.DistanceM, seedLogo); err != nil {
			log.Fatalf("[ERROR] menambah hotel %q: %v", item.Name, err)
		}
		created++
	}
	return
}

func seedAirlines(ctx context.Context, tx *sql.Tx) (created, updated, skipped int) {
	for _, item := range airlines {
		var id uint64
		var code sql.NullString
		var logoURL sql.NullString
		nameUpper := strings.ToUpper(strings.TrimSpace(item.Name))
		err := tx.QueryRowContext(ctx, `SELECT id, code, logo_url FROM airlines WHERE UPPER(name)=? LIMIT 1`, nameUpper).Scan(&id, &code, &logoURL)
		if err == nil {
			needUpdate := false
			newCode := code.String
			newLogo := logoURL.String

			if item.Code != "" && (!code.Valid || code.String == "") {
				newCode = item.Code
				needUpdate = true
			}
			if item.LogoURL != "" && (!logoURL.Valid || logoURL.String == "") {
				newLogo = item.LogoURL
				needUpdate = true
			}

			if needUpdate {
				var finalCode any
				if newCode != "" {
					finalCode = newCode
				}
				var finalLogo any
				if newLogo != "" {
					finalLogo = newLogo
				}
				if _, err := tx.ExecContext(ctx, `UPDATE airlines SET code=?, logo_url=? WHERE id=?`, finalCode, finalLogo, id); err != nil {
					log.Fatalf("[ERROR] memperbarui maskapai %q: %v", item.Name, err)
				}
				updated++
				continue
			}
			skipped++
			continue
		}
		if err != sql.ErrNoRows {
			log.Fatalf("[ERROR] memeriksa maskapai %q: %v", item.Name, err)
		}

		var seedCode any
		if item.Code != "" {
			seedCode = item.Code
		}
		var seedLogo any
		if item.LogoURL != "" {
			seedLogo = item.LogoURL
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO airlines (name, code, logo_url) VALUES (?, ?, ?)`, nameUpper, seedCode, seedLogo); err != nil {
			log.Fatalf("[ERROR] menambah maskapai %q: %v", item.Name, err)
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
