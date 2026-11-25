package adminuser

// Integration test aturan Pengguna (SPEK-AKSES-PENGGUNA-2026-11-25 §2).
// Repository membuka transaksinya sendiri, jadi data uji dibuat sungguhan
// lalu dibersihkan (email gotest.au.*@local.test; awalan per paket karena
// paket uji berjalan paralel). Jalan hanya dengan
// ERP_TEST_DB=1:
//
//	$env:ERP_TEST_DB = '1'; go test ./internal/adminuser/ -run Integrasi -v

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/akses"
	"erp-azhan/api/internal/testdb"
)

const emailUji = "gotest.au.%@local.test"

type fixture struct {
	t    *testing.T
	db   *sql.DB
	repo *Repository
	ctx  context.Context
}

func setup(t *testing.T) *fixture {
	t.Helper()
	db := testdb.DB(t)
	testdb.Serialize(t, db, "erp_test_pengguna")
	clean := func() {
		if _, err := db.Exec(`DELETE FROM admin_users WHERE email LIKE ?`, emailUji); err != nil {
			t.Errorf("bersihkan pengguna uji: %v", err)
		}
	}
	clean()
	t.Cleanup(clean)
	return &fixture{t: t, db: db, repo: NewRepository(db), ctx: context.Background()}
}

func (f *fixture) brand() *uint64 {
	f.t.Helper()
	var id uint64
	if err := f.db.QueryRow(`SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&id); err != nil {
		f.t.Fatalf("brand: %v", err)
	}
	return &id
}

func (f *fixture) count(q string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRow(q, args...).Scan(&n); err != nil {
		f.t.Fatalf("count %q: %v", q, err)
	}
	return n
}

func (f *fixture) create(in UpsertInput) uint64 {
	f.t.Helper()
	id, err := f.repo.Create(f.ctx, in, "hash-uji")
	if err != nil {
		f.t.Fatalf("create %s: %v", in.Email, err)
	}
	return id
}

func (f *fixture) sesi(id uint64) {
	f.t.Helper()
	if _, err := f.db.Exec(`INSERT INTO admin_refresh_tokens (jti, admin_user_id, expires_at)
		VALUES (UUID(), ?, NOW() + INTERVAL 1 DAY)`, id); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) sesiAktif(id uint64) int {
	return f.count(`SELECT COUNT(*) FROM admin_refresh_tokens WHERE admin_user_id = ? AND revoked_at IS NULL`, id)
}

func (f *fixture) versi(id uint64) int {
	return f.count(`SELECT perm_version FROM admin_users WHERE id = ?`, id)
}

// tim membuat input pengguna Tim dengan izin yang diberikan.
func tim(email string, brandID *uint64, izin ...string) UpsertInput {
	return UpsertInput{DisplayName: "Go Test", Email: email, BrandID: brandID, Akses: akses.Tim, Permissions: izin}
}

func superAdmin(email string, brandID *uint64) UpsertInput {
	return UpsertInput{DisplayName: "Go Test", Email: email, BrandID: brandID, Akses: akses.SuperAdmin}
}

func TestIntegrasiBuatTimMenyimpanIzin(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t1@local.test", f.brand(), "booking.view", "booking.edit", "booking.view"))

	u, err := f.repo.GetByID(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if u.Akses != akses.Tim {
		t.Errorf("akses = %q, harap tim", u.Akses)
	}
	if len(u.Permissions) != 2 || u.Permissions[0] != "booking.edit" || u.Permissions[1] != "booking.view" {
		t.Errorf("izin = %v, harap [booking.edit booking.view]", u.Permissions)
	}
}

func TestIntegrasiSuperAdminTanpaIzinTersimpan(t *testing.T) {
	f := setup(t)
	in := superAdmin("gotest.au.sa@local.test", nil)
	in.Permissions = []string{"booking.view"}
	id := f.create(in)

	u, err := f.repo.GetByID(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if u.Akses != akses.SuperAdmin || len(u.Permissions) != 0 {
		t.Errorf("akses = %q izin = %v, harap super_admin tanpa izin tersimpan", u.Akses, u.Permissions)
	}
	if n := f.count(`SELECT COUNT(*) FROM user_permissions WHERE admin_user_id = ?`, id); n != 0 {
		t.Errorf("user_permissions = %d, harap 0", n)
	}
	// Klaim token Super Admin memuat seluruh izin.
	acc, err := akses.NewRepository(f.db).GetUserAccess(f.ctx, int64(id))
	if err != nil {
		t.Fatal(err)
	}
	if semua := f.count(`SELECT COUNT(*) FROM permissions`); acc.Jenis != akses.SuperAdmin || len(acc.Permissions) != semua {
		t.Errorf("klaim: jenis=%q izin=%d, harap super_admin dengan %d izin", acc.Jenis, len(acc.Permissions), semua)
	}
}

// Create yang ditolak tidak boleh meninggalkan baris (transaksi rollback).
func TestIntegrasiAksesTakSahTidakMeninggalkanData(t *testing.T) {
	f := setup(t)
	cases := []struct {
		nama string
		in   UpsertInput
		want error
	}{
		{"super admin terikat brand", superAdmin("gotest.au.x1@local.test", f.brand()), akses.ErrSuperAdminBrand},
		{"tim tanpa izin", tim("gotest.au.x2@local.test", nil), akses.ErrTanpaIzin},
		{"izin khusus super admin", tim("gotest.au.x3@local.test", nil, "booking.view", "user.admin"), akses.ErrIzinTidakSah},
		{"izin holding untuk brand", tim("gotest.au.x4@local.test", f.brand(), "booking.view", "schedule.edit"), akses.ErrIzinTidakSah},
		{"kode tidak dikenal", tim("gotest.au.x5@local.test", nil, "tidak.ada"), akses.ErrIzinTidakSah},
		{"jenis akses salah", UpsertInput{DisplayName: "Go Test", Email: "gotest.au.x6@local.test", Akses: "admin"}, akses.ErrJenisTidakSah},
	}
	for _, c := range cases {
		if _, err := f.repo.Create(f.ctx, c.in, "hash-uji"); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, harap %v", c.nama, err, c.want)
		}
		if n := f.count(`SELECT COUNT(*) FROM admin_users WHERE email = ?`, c.in.Email); n != 0 {
			t.Errorf("%s: baris tercipta = %d, harap 0", c.nama, n)
		}
	}
	if n := f.count(`SELECT COUNT(*) FROM user_permissions up JOIN admin_users u ON u.id = up.admin_user_id
		WHERE u.email LIKE ?`, emailUji); n != 0 {
		t.Errorf("izin tersisa = %d, harap 0", n)
	}
}

// Pindah ke akses brand divalidasi terhadap brand BARU; yang ditolak tidak mengubah apa pun.
func TestIntegrasiUbahKeBrandDenganIzinHoldingDitolak(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t3@local.test", nil, "schedule.edit"))
	err := f.repo.Update(f.ctx, id, tim("gotest.au.t3@local.test", f.brand(), "schedule.edit"))
	if !errors.Is(err, akses.ErrIzinTidakSah) {
		t.Fatalf("err = %v, harap ErrIzinTidakSah", err)
	}
	if n := f.count(`SELECT COUNT(*) FROM admin_users WHERE id = ? AND brand_id IS NULL`, id); n != 1 {
		t.Error("akses brand berubah padahal update ditolak")
	}
}

func TestIntegrasiUbahIzinMencabutSesi(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t5@local.test", nil, "booking.view"))
	f.sesi(id)
	awal := f.versi(id)

	// Hanya profil yang berubah: sesi tetap, versi tetap.
	sama := tim("gotest.au.t5@local.test", nil, "booking.view")
	sama.DisplayName = "Go Test Dua"
	if err := f.repo.Update(f.ctx, id, sama); err != nil {
		t.Fatal(err)
	}
	if f.sesiAktif(id) != 1 || f.versi(id) != awal {
		t.Fatalf("ubah profil saja: sesi=%d versi=%d, harap sesi 1 versi %d", f.sesiAktif(id), f.versi(id), awal)
	}

	if err := f.repo.Update(f.ctx, id, tim("gotest.au.t5@local.test", nil, "booking.view", "jamaah.view")); err != nil {
		t.Fatal(err)
	}
	if f.sesiAktif(id) != 0 || f.versi(id) != awal+1 {
		t.Fatalf("ubah izin: sesi=%d versi=%d, harap sesi 0 versi %d", f.sesiAktif(id), f.versi(id), awal+1)
	}
	u, _ := f.repo.GetByID(f.ctx, id)
	if len(u.Permissions) != 2 {
		t.Errorf("izin = %v, harap 2 izin", u.Permissions)
	}

	// Tim menjadi Super Admin: izin tersimpan dikosongkan.
	if err := f.repo.Update(f.ctx, id, superAdmin("gotest.au.t5@local.test", nil)); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM user_permissions WHERE admin_user_id = ?`, id); n != 0 {
		t.Errorf("izin tersisa setelah jadi Super Admin = %d, harap 0", n)
	}
}

func TestIntegrasiNonaktifkanDiriSendiri(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t6@local.test", nil, "booking.view"))
	if err := f.repo.SetStatus(f.ctx, id, false, id); !errors.Is(err, ErrCannotDeactivateSelf) {
		t.Fatalf("err = %v, harap ErrCannotDeactivateSelf", err)
	}
	if err := f.repo.Delete(f.ctx, id, id); !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("err = %v, harap ErrCannotDeleteSelf", err)
	}
}

func TestIntegrasiNonaktifMencabutSesi(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t7@local.test", nil, "booking.view"))
	f.sesi(id)
	verSebelum := f.versi(id)

	if err := f.repo.SetStatus(f.ctx, id, false, 0); err != nil {
		t.Fatal(err)
	}
	if n := f.sesiAktif(id); n != 0 {
		t.Errorf("refresh token aktif = %d, harap 0", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM admin_users WHERE id = ? AND is_active = FALSE AND perm_version > ?`, id, verSebelum); n != 1 {
		t.Error("pengguna harus nonaktif dan perm_version naik")
	}
}

func TestIntegrasiHapusPenggunaBerRiwayat(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t8@local.test", nil, "booking.view"))
	var jamaahID int64
	if err := f.db.QueryRow(`SELECT id FROM jamaah WHERE disetujui_agen_oleh IS NULL ORDER BY id LIMIT 1`).Scan(&jamaahID); err != nil {
		t.Skipf("tidak ada jamaah untuk fixture riwayat: %v", err)
	}
	if _, err := f.db.Exec(`UPDATE jamaah SET disetujui_agen_oleh = ? WHERE id = ?`, id, jamaahID); err != nil {
		t.Fatal(err)
	}
	// LIFO: dipulihkan sebelum pembersihan pengguna uji.
	t.Cleanup(func() { _, _ = f.db.Exec(`UPDATE jamaah SET disetujui_agen_oleh = NULL WHERE id = ?`, jamaahID) })

	if err := f.repo.Delete(f.ctx, id, 0); !errors.Is(err, ErrHasHistory) {
		t.Fatalf("err = %v, harap ErrHasHistory", err)
	}
}

func TestIntegrasiHapusPenggunaTanpaRiwayat(t *testing.T) {
	f := setup(t)
	id := f.create(tim("gotest.au.t9@local.test", nil, "booking.view"))
	if err := f.repo.Delete(f.ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM admin_users WHERE id = ?`,
		`SELECT COUNT(*) FROM user_permissions WHERE admin_user_id = ?`,
	} {
		if n := f.count(q, id); n != 0 {
			t.Errorf("%s = %d, harap 0", q, n)
		}
	}
}
