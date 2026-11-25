package akses

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestNormalize(t *testing.T) {
	got := Normalize([]string{" booking.view ", "agen.view", "booking.view", "  "})
	if len(got) != 2 || got[0] != "agen.view" || got[1] != "booking.view" {
		t.Fatalf("Normalize = %v, harap [agen.view booking.view]", got)
	}
}

func TestValidasiJenis(t *testing.T) {
	brand := int64(1)
	cases := []struct {
		jenis string
		brand *int64
		want  error
	}{
		{SuperAdmin, nil, nil},
		{SuperAdmin, &brand, ErrSuperAdminBrand},
		{Tim, nil, nil},
		{Tim, &brand, nil},
		{"admin", nil, ErrJenisTidakSah},
		{"", nil, ErrJenisTidakSah},
	}
	for _, c := range cases {
		if err := ValidasiJenis(c.jenis, c.brand); !errors.Is(err, c.want) {
			t.Errorf("ValidasiJenis(%q, brand=%v) = %v, harap %v", c.jenis, c.brand != nil, err, c.want)
		}
	}
}

// Penolakan yang tidak butuh database: berhenti sebelum query.
func TestValidasiIzinTimMenolakTanpaDB(t *testing.T) {
	brand := int64(1)
	ctx := context.Background()
	if err := ValidasiIzinTim(ctx, nil, nil, nil); !errors.Is(err, ErrTanpaIzin) {
		t.Errorf("tanpa izin: %v", err)
	}
	for kode := range hanyaSuperAdmin {
		if err := ValidasiIzinTim(ctx, nil, []string{kode}, nil); !errors.Is(err, ErrIzinTidakSah) {
			t.Errorf("%s untuk Tim: %v, harap ErrIzinTidakSah", kode, err)
		}
		if izinBrand[kode] {
			t.Errorf("%s khusus Super Admin tidak boleh ada di izinBrand", kode)
		}
	}
	for _, kode := range []string{"schedule.edit", "masterdata.edit", "cms.edit", "agen.payout.approve", "booking.diskon.besar"} {
		if err := ValidasiIzinTim(ctx, nil, []string{kode}, &brand); !errors.Is(err, ErrIzinTidakSah) {
			t.Errorf("%s untuk pengguna brand: %v, harap ErrIzinTidakSah", kode, err)
		}
	}
}

// Setiap kode di daftar tetap harus ada di tabel permissions: salah ketik di sini
// akan membuat izin itu tidak pernah bisa diberikan atau tidak pernah dibatasi.
func TestDaftarIzinAdaDiDatabase(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	ada := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT code FROM permissions`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		ada[c] = true
	}
	rows.Close()
	for kode := range izinBrand {
		if !ada[kode] {
			t.Errorf("izinBrand memuat %q yang tidak ada di tabel permissions", kode)
		}
	}
	for kode := range hanyaSuperAdmin {
		if !ada[kode] {
			t.Errorf("hanyaSuperAdmin memuat %q yang tidak ada di tabel permissions", kode)
		}
	}
	if err := ValidasiIzinTim(ctx, tx, []string{"booking.view", "tidak.ada"}, nil); !errors.Is(err, ErrIzinTidakSah) {
		t.Errorf("kode tidak dikenal: %v, harap ErrIzinTidakSah", err)
	}
	if err := ValidasiIzinTim(ctx, tx, []string{"booking.view", "schedule.edit"}, nil); err != nil {
		t.Errorf("izin holding sah ditolak: %v", err)
	}
}

// A6: Super Admin aktif terakhir tidak boleh hilang.
func TestSuperAdminTerakhirDijaga(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	var sisa int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM admin_users WHERE is_super_admin = TRUE AND is_active = TRUE ORDER BY id LIMIT 1`).Scan(&sisa); err != nil {
		t.Skipf("tidak ada Super Admin aktif: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE admin_users SET is_active = FALSE WHERE is_super_admin = TRUE AND id <> ?`, sisa); err != nil {
		t.Fatal(err)
	}
	if err := EnsureOtherActiveSuperAdmin(ctx, tx, sisa); !errors.Is(err, ErrLastSuperAdmin) {
		t.Fatalf("Super Admin terakhir: err = %v, harap ErrLastSuperAdmin", err)
	}
	if err := EnsureOtherActiveSuperAdmin(ctx, tx, 0); err != nil {
		t.Fatalf("masih ada Super Admin lain: err = %v, harap nil", err)
	}
}
