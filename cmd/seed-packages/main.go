package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type PackageRecord struct {
	Brand                    string   `json:"brand"`
	JadwalNama               string   `json:"jadwal_nama"`
	IsPromo                  bool     `json:"is_promo"`
	SeatTotal                int      `json:"seat_total"`
	SeatSisa                 int      `json:"seat_sisa"`
	Maskapai                 string   `json:"maskapai"`
	BerangkatTanggal         string   `json:"berangkat_tanggal"`
	BerangkatJam             string   `json:"berangkat_jam"`
	BerangkatKodePenerbangan string   `json:"berangkat_kode_penerbangan"`
	PulangTanggal            string   `json:"pulang_tanggal"`
	PulangJam                string   `json:"pulang_jam"`
	PulangKodePenerbangan    string   `json:"pulang_kode_penerbangan"`
	HotelMekkah              string   `json:"hotel_mekkah"`
	HotelMadinah             string   `json:"hotel_madinah"`
	HargaQuad                int      `json:"harga_quad"`
	HargaTriple              int      `json:"harga_triple"`
	HargaDouble              int      `json:"harga_double"`
	HargaCoret               *int     `json:"harga_coret"`
	ItineraryID              *int     `json:"itinerary_id"`
	IncludeItems             []string `json:"include_items"`
	ExcludeItems             []string `json:"exclude_items"`
	Status                   string   `json:"status"`
	IsTicketConfirmed        bool     `json:"is_ticket_confirmed"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/seed-packages/main.go <seed_file.json>")
		os.Exit(1)
	}

	seedFile := os.Args[1]

	data, err := os.ReadFile(seedFile)
	if err != nil {
		fmt.Printf("[ERROR] Gagal membaca file %s: %v\n", seedFile, err)
		os.Exit(1)
	}

	var records []PackageRecord
	if err := json.Unmarshal(data, &records); err != nil {
		fmt.Printf("[ERROR] Gagal parse JSON: %v\n", err)
		os.Exit(1)
	}

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

	totalProcessed := 0
	totalSuccess := 0
	var errorSkips []string
	var newHotels []string
	var newAirlines []string

	for _, rec := range records {
		totalProcessed++

		// 1. Resolve brand_id
		var brandID int
		err := db.QueryRow("SELECT id FROM brands WHERE name = ?", rec.Brand).Scan(&brandID)
		if err != nil {
			errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: brand tidak ditemukan: %s", totalProcessed, rec.Brand))
			continue
		}

		// 2. Resolve/create maskapai
		var airlineID int
		err = db.QueryRow("SELECT id FROM airlines WHERE UPPER(name) = UPPER(?)", rec.Maskapai).Scan(&airlineID)
		if err != nil {
			if err == sql.ErrNoRows {
				res, err := db.Exec("INSERT INTO airlines (name) VALUES (?)", strings.ToUpper(rec.Maskapai))
				if err != nil {
					errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: gagal insert airline %s: %v", totalProcessed, rec.Maskapai, err))
					continue
				}
				id, _ := res.LastInsertId()
				airlineID = int(id)
				newAirlines = append(newAirlines, strings.ToUpper(rec.Maskapai))
			} else {
				errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: error query airline %s: %v", totalProcessed, rec.Maskapai, err))
				continue
			}
		}

		// 3. Resolve/create hotel mekkah
		var hotelMekkahID int
		err = db.QueryRow("SELECT id FROM hotels WHERE UPPER(name) = UPPER(?)", rec.HotelMekkah).Scan(&hotelMekkahID)
		if err != nil {
			if err == sql.ErrNoRows {
				res, err := db.Exec("INSERT INTO hotels (name, city) VALUES (?, 'mekkah')", strings.ToUpper(rec.HotelMekkah))
				if err != nil {
					errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: gagal insert hotel mekkah %s: %v", totalProcessed, rec.HotelMekkah, err))
					continue
				}
				id, _ := res.LastInsertId()
				hotelMekkahID = int(id)
				newHotels = append(newHotels, strings.ToUpper(rec.HotelMekkah))
			} else {
				errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: error query hotel mekkah %s: %v", totalProcessed, rec.HotelMekkah, err))
				continue
			}
		}

		// 4. Resolve/create hotel madinah
		var hotelMadinahID int
		err = db.QueryRow("SELECT id FROM hotels WHERE UPPER(name) = UPPER(?)", rec.HotelMadinah).Scan(&hotelMadinahID)
		if err != nil {
			if err == sql.ErrNoRows {
				res, err := db.Exec("INSERT INTO hotels (name, city) VALUES (?, 'madinah')", strings.ToUpper(rec.HotelMadinah))
				if err != nil {
					errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: gagal insert hotel madinah %s: %v", totalProcessed, rec.HotelMadinah, err))
					continue
				}
				id, _ := res.LastInsertId()
				hotelMadinahID = int(id)
				newHotels = append(newHotels, strings.ToUpper(rec.HotelMadinah))
			} else {
				errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: error query hotel madinah %s: %v", totalProcessed, rec.HotelMadinah, err))
				continue
			}
		}

		// 5. INSERT ke schedules
		includeJSON, _ := json.Marshal(rec.IncludeItems)
		if len(rec.IncludeItems) == 0 {
			includeJSON = []byte("[]")
		}
		excludeJSON, _ := json.Marshal(rec.ExcludeItems)
		if len(rec.ExcludeItems) == 0 {
			excludeJSON = []byte("[]")
		}

		var bJam interface{} = rec.BerangkatJam
		if rec.BerangkatJam == "" {
			bJam = nil
		}
		var bKode interface{} = rec.BerangkatKodePenerbangan
		if rec.BerangkatKodePenerbangan == "" {
			bKode = nil
		}
		var pJam interface{} = rec.PulangJam
		if rec.PulangJam == "" {
			pJam = nil
		}
		var pKode interface{} = rec.PulangKodePenerbangan
		if rec.PulangKodePenerbangan == "" {
			pKode = nil
		}

		_, err = db.Exec(`
			INSERT INTO schedules (
				brand_id, maskapai_id, hotel_mekkah_id, hotel_madinah_id,
				jadwal_nama, is_promo, seat_total, seat_sisa,
				berangkat_tanggal, berangkat_jam, berangkat_kode_penerbangan,
				pulang_tanggal, pulang_jam, pulang_kode_penerbangan,
				harga_quad, harga_triple, harga_double, harga_coret,
				itinerary_id, include_items, exclude_items, status, is_ticket_confirmed
			) VALUES (
				?, ?, ?, ?,
				?, ?, ?, ?,
				?, ?, ?,
				?, ?, ?,
				?, ?, ?, ?,
				?, ?, ?, 'draft', false
			)
		`,
			brandID, airlineID, hotelMekkahID, hotelMadinahID,
			rec.JadwalNama, rec.IsPromo, rec.SeatTotal, rec.SeatSisa,
			rec.BerangkatTanggal, bJam, bKode,
			rec.PulangTanggal, pJam, pKode,
			rec.HargaQuad, rec.HargaTriple, rec.HargaDouble, rec.HargaCoret,
			rec.ItineraryID, string(includeJSON), string(excludeJSON),
		)

		if err != nil {
			errorSkips = append(errorSkips, fmt.Sprintf("Baris %d: gagal insert schedule: %v", totalProcessed, err))
			continue
		}

		totalSuccess++
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("RINGKASAN SEEDING")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total record diproses: %d\n", totalProcessed)
	fmt.Printf("Total sukses insert: %d\n", totalSuccess)
	fmt.Printf("Total gagal/skip: %d\n", len(errorSkips))
	if len(errorSkips) > 0 {
		for _, e := range errorSkips {
			fmt.Println(" -", e)
		}
	}
	fmt.Printf("Total hotel baru dibuat: %d\n", len(newHotels))
	if len(newHotels) > 0 {
		for _, h := range newHotels {
			fmt.Println(" -", h)
		}
	}
	fmt.Printf("Total maskapai baru dibuat: %d\n", len(newAirlines))
	if len(newAirlines) > 0 {
		for _, a := range newAirlines {
			fmt.Println(" -", a)
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
