// Command test_token mencetak access token admin untuk uji lokal.
//
// Hanya untuk pengembangan: token ditandatangani JWT_SECRET dari .env, jadi bila dijalankan dengan .env
// produksi, token ini sah di server produksi. Karena itu perintah ini menolak berjalan bila
// APP_ENV=production (audit keamanan 2026-11-12).
package main

import (
	"fmt"
	"log"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	if shared.IsProduction() {
		log.Fatal("test_token menolak berjalan dengan APP_ENV=production: token yang dibuat akan sah di server produksi")
	}
	brandID := int64(2)
	acc, err := identity.GenerateAccessToken(8, &brandID, "admin")
	if err != nil {
		log.Fatalf("gagal membuat token: %v", err)
	}
	fmt.Println(acc)
}
