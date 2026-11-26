package payment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/testdb"
)

// Bukti yang dicatat admin Holding harus unggahan Holding atau unggahan brand booking itu: bukti
// yang ditautkan ke booking bisa dibuka admin brand booking (audit akses 2026-11-26, T5).
func TestValidasiBuktiAdminHoldingSesuaiBrandBooking(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	for _, stmt := range []string{
		`CREATE TEMPORARY TABLE schedules(id BIGINT UNSIGNED PRIMARY KEY, brand_id BIGINT UNSIGNED NOT NULL)`,
		`CREATE TEMPORARY TABLE bookings(id BIGINT UNSIGNED PRIMARY KEY, schedule_id BIGINT UNSIGNED NOT NULL)`,
		`INSERT INTO schedules VALUES (1, 10), (2, 20)`,
		`INSERT INTO bookings VALUES (100, 1), (200, 2)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join("uploads", "payment-proofs"), 0700); err != nil {
		t.Fatal(err)
	}
	nama := map[string]string{
		"brand10": "11111111-1111-1111-1111-111111111111.pdf",
		"brand20": "22222222-2222-2222-2222-222222222222.pdf",
		"holding": "33333333-3333-3333-3333-333333333333.pdf",
	}
	for _, n := range nama {
		if err := os.WriteFile(filepath.Join("uploads", "payment-proofs", n), []byte("%PDF-1.4"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	url := func(k string) string { return "/api/admin/media/payment-proofs/" + nama[k] }
	if _, err := db.Exec(`INSERT INTO media_uploads(path,brand_id,admin_user_id) VALUES(?,10,5),(?,20,6),(?,NULL,1)`,
		url("brand10"), url("brand20"), url("holding")); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	ctx := context.Background()
	for _, c := range []struct {
		nama    string
		bukti   string
		booking int64
		sah     bool
	}{
		{"berkas brand 10, booking brand 10", url("brand10"), 100, true},
		{"berkas Holding, booking brand 10", url("holding"), 100, true},
		{"bentuk lama /uploads/", "/uploads/payment-proofs/" + nama["holding"], 200, true},
		{"berkas brand 20, booking brand 10", url("brand20"), 100, false},
		{"berkas brand 10, booking brand 20", url("brand10"), 200, false},
	} {
		err := repo.ValidasiBuktiAdmin(ctx, c.bukti, nil, c.booking)
		var ue *media.UploadError
		switch {
		case c.sah && err != nil:
			t.Errorf("%s ditolak: %v", c.nama, err)
		case !c.sah && !errors.As(err, &ue):
			t.Errorf("%s: err=%v, harap UploadError", c.nama, err)
		}
	}
}
