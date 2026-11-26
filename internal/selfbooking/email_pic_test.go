package selfbooking

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit booking 2026-11-26: email akun pemesan hanya melengkapi jamaah milik akun itu (dibuat akun itu atau
// tertaut lewat aktivasi), bukan jamaah orang lain yang cocok lewat NIK + nama.
func TestEmailPICHanyaUntukJamaahMilikAkun(t *testing.T) {
	tx, _, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemesan Email"})
	email := "pemesan-uji@example.com"
	emailJamaah := func(id int64) string {
		var v sql.NullString
		if err := tx.QueryRow(`SELECT email FROM jamaah WHERE id=?`, id).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v.String
	}

	// Jamaah lama milik orang lain (dibuat admin/agen), cocok lewat NIK + nama: email tidak diisi.
	nikLain := nikUji()
	lama := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Orang Lain", NIK: nikLain})
	id, err := resolvePIC(ctx, tx, brand, akun, PICInput{NamaLengkap: "Orang Lain (uji)", JenisKelamin: "L", NIK: nikp(nikLain), Email: &email})
	if err != nil {
		t.Fatalf("resolve PIC jamaah lama: %v", err)
	}
	if id != lama {
		t.Fatalf("PIC = jamaah %d, want jamaah lama %d", id, lama)
	}
	if got := emailJamaah(lama); got != "" {
		t.Fatalf("email jamaah orang lain = %q, want tetap kosong", got)
	}

	// Jamaah yang sama setelah tertaut ke akun lewat aktivasi: email dilengkapi.
	if _, err := tx.Exec(`UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, lama, akun); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvePIC(ctx, tx, brand, akun, PICInput{NamaLengkap: "Orang Lain (uji)", JenisKelamin: "L", NIK: nikp(nikLain), Email: &email}); err != nil {
		t.Fatalf("resolve PIC jamaah tertaut: %v", err)
	}
	if got := emailJamaah(lama); got != email {
		t.Fatalf("email jamaah tertaut = %q, want %q", got, email)
	}

	// Jamaah baru yang dibuat di booking ini: email dilengkapi.
	baru, err := resolvePIC(ctx, tx, brand, akun, PICInput{NamaLengkap: "Jamaah Baru Email", JenisKelamin: "P", NIK: nikp(nikUji()), Email: &email})
	if err != nil {
		t.Fatalf("resolve PIC jamaah baru: %v", err)
	}
	if got := emailJamaah(baru); got != email {
		t.Fatalf("email jamaah baru = %q, want %q", got, email)
	}
}
