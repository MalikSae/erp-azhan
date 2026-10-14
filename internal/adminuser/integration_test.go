package adminuser

// Integration test aturan Pengguna (SPEK-PENGGUNA-ORGANISASI §4 & §10).
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

	"erp-azhan/api/internal/rbac"
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

func (f *fixture) unit(nama string) int64 {
	f.t.Helper()
	var id int64
	if err := f.db.QueryRow(`SELECT id FROM org_unit WHERE nama = ?`, nama).Scan(&id); err != nil {
		f.t.Fatalf("unit %q: %v", nama, err)
	}
	return id
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

func pengguna(email string, brandID *uint64, roles []string, k ...KeanggotaanInput) UpsertInput {
	return UpsertInput{DisplayName: "Go Test", Email: email, BrandID: brandID, Roles: roles, Keanggotaan: k}
}

func tugas(unitID int64, posisi string) KeanggotaanInput {
	return KeanggotaanInput{OrgUnitID: unitID, Posisi: posisi}
}

func TestIntegrasiT1BuatLengkap(t *testing.T) {
	f := setup(t)
	mg, bc := f.unit("Marketing & Growth"), f.unit("Brand & Community")
	id := f.create(pengguna("gotest.au.t1@local.test", nil, []string{"cmo"},
		tugas(mg, "pemimpin"), tugas(bc, "anggota")))

	u, err := f.repo.GetByID(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(u.Roles) != 1 || u.Roles[0] != "cmo" {
		t.Errorf("roles = %v, harap [cmo]", u.Roles)
	}
	if len(u.Keanggotaan) != 2 {
		t.Errorf("keanggotaan = %d, harap 2", len(u.Keanggotaan))
	}
	if n := f.count(`SELECT COUNT(*) FROM org_member WHERE admin_user_id = ? AND org_unit_id = ? AND posisi = 'pemimpin'`, id, mg); n != 1 {
		t.Errorf("pemimpin M&G = %d, harap 1", n)
	}
}

func TestIntegrasiT2ScopeSalahTidakMeninggalkanData(t *testing.T) {
	f := setup(t)
	_, err := f.repo.Create(f.ctx, pengguna("gotest.au.t2@local.test", f.brand(), []string{"direktur"}), "hash-uji")
	if !errors.Is(err, rbac.ErrScopeMismatch) {
		t.Fatalf("err = %v, harap ErrScopeMismatch", err)
	}
	if n := f.count(`SELECT COUNT(*) FROM admin_users WHERE email = 'gotest.au.t2@local.test'`); n != 0 {
		t.Errorf("baris tercipta = %d, harap 0 (transaksi harus rollback)", n)
	}
}

func TestIntegrasiT3UbahAksesMembuatRoleTakSah(t *testing.T) {
	f := setup(t)
	id := f.create(pengguna("gotest.au.t3@local.test", nil, []string{"direktur"}))
	err := f.repo.Update(f.ctx, id, pengguna("gotest.au.t3@local.test", f.brand(), []string{"direktur"}))
	if !errors.Is(err, rbac.ErrScopeMismatch) {
		t.Fatalf("err = %v, harap ErrScopeMismatch", err)
	}
	if n := f.count(`SELECT COUNT(*) FROM admin_users WHERE id = ? AND brand_id IS NULL`, id); n != 1 {
		t.Error("akses brand berubah padahal update ditolak")
	}
}

func TestIntegrasiT4DuaPemimpinDanDuaTim(t *testing.T) {
	f := setup(t)
	seo, cp := f.unit("SEO"), f.unit("Creative Production")
	f.create(pengguna("gotest.au.t4a@local.test", nil, []string{"marketing_lead"}, tugas(seo, "pemimpin"), tugas(cp, "anggota")))
	f.create(pengguna("gotest.au.t4b@local.test", nil, []string{"marketing_lead"}, tugas(seo, "pemimpin")))
	if n := f.count(`SELECT COUNT(*) FROM org_member m JOIN admin_users u ON u.id = m.admin_user_id
		WHERE m.org_unit_id = ? AND m.posisi = 'pemimpin' AND u.email LIKE ?`, seo, emailUji); n != 2 {
		t.Errorf("pemimpin SEO = %d, harap 2", n)
	}
}

func TestIntegrasiT6NonaktifkanDiriSendiri(t *testing.T) {
	f := setup(t)
	id := f.create(pengguna("gotest.au.t6@local.test", nil, []string{"direktur"}))
	if err := f.repo.SetStatus(f.ctx, id, false, id); !errors.Is(err, ErrCannotDeactivateSelf) {
		t.Fatalf("err = %v, harap ErrCannotDeactivateSelf", err)
	}
	if err := f.repo.Delete(f.ctx, id, id); !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("err = %v, harap ErrCannotDeleteSelf", err)
	}
}

func TestIntegrasiT7NonaktifMencabutSesi(t *testing.T) {
	f := setup(t)
	id := f.create(pengguna("gotest.au.t7@local.test", nil, []string{"direktur"}))
	if _, err := f.db.Exec(`INSERT INTO admin_refresh_tokens (jti, admin_user_id, expires_at)
		VALUES (UUID(), ?, NOW() + INTERVAL 1 DAY)`, id); err != nil {
		t.Fatal(err)
	}
	var verSebelum int
	_ = f.db.QueryRow(`SELECT perm_version FROM admin_users WHERE id = ?`, id).Scan(&verSebelum)

	if err := f.repo.SetStatus(f.ctx, id, false, 0); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM admin_refresh_tokens WHERE admin_user_id = ? AND revoked_at IS NULL`, id); n != 0 {
		t.Errorf("refresh token aktif = %d, harap 0", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM admin_users WHERE id = ? AND is_active = FALSE AND perm_version > ?`, id, verSebelum); n != 1 {
		t.Error("pengguna harus nonaktif dan perm_version naik")
	}
}

func TestIntegrasiT8HapusPenggunaBerRiwayat(t *testing.T) {
	f := setup(t)
	id := f.create(pengguna("gotest.au.t8@local.test", nil, []string{"direktur"}))
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

func TestIntegrasiT9HapusPenggunaTanpaRiwayat(t *testing.T) {
	f := setup(t)
	id := f.create(pengguna("gotest.au.t9@local.test", nil, []string{"direktur"}, tugas(f.unit("Manajemen"), "anggota")))
	if err := f.repo.Delete(f.ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM admin_users WHERE id = ?`,
		`SELECT COUNT(*) FROM user_roles WHERE admin_user_id = ?`,
		`SELECT COUNT(*) FROM org_member WHERE admin_user_id = ?`,
	} {
		if n := f.count(q, id); n != 0 {
			t.Errorf("%s = %d, harap 0", q, n)
		}
	}
}

func TestIntegrasiKolomRoleLamaUntukGatewayCRM(t *testing.T) {
	f := setup(t)
	cases := []struct {
		email string
		roles []string
		want  string
	}{
		{"gotest.au.cs1@local.test", []string{"cs"}, "cs"},
		{"gotest.au.cs2@local.test", []string{"cs", "admin_cabang"}, "admin"},
	}
	for _, c := range cases {
		id := f.create(pengguna(c.email, f.brand(), c.roles))
		var role string
		_ = f.db.QueryRow(`SELECT role FROM admin_users WHERE id = ?`, id).Scan(&role)
		if role != c.want {
			t.Errorf("%v: role lama = %q, harap %q", c.roles, role, c.want)
		}
	}
}

func TestIntegrasiR10PenugasanMengikutiRole(t *testing.T) {
	f := setup(t)
	mg, ku, seo, it := f.unit("Marketing & Growth"), f.unit("Keuangan"), f.unit("SEO"), f.unit("IT")

	ditolak := []struct {
		nama string
		in   UpsertInput
	}{
		{"Finance di Marketing", pengguna("gotest.au.r1@local.test", nil, []string{"finance"}, tugas(mg, "anggota"))},
		{"Direktur di Keuangan", pengguna("gotest.au.r2@local.test", nil, []string{"direktur"}, tugas(ku, "anggota"))},
		{"akses Brand berpenugasan", pengguna("gotest.au.r3@local.test", f.brand(), []string{"finance"}, tugas(ku, "anggota"))},
	}
	for _, c := range ditolak {
		if _, err := f.repo.Create(f.ctx, c.in, "hash-uji"); !errors.Is(err, rbac.ErrUnitMismatch) {
			t.Errorf("%s: err = %v, harap ErrUnitMismatch", c.nama, err)
		}
	}

	f.create(pengguna("gotest.au.r4@local.test", nil, []string{"direktur", "finance"}, tugas(f.unit("Manajemen"), "pemimpin"), tugas(ku, "pemimpin")))
	f.create(pengguna("gotest.au.r5@local.test", nil, []string{rbac.SuperAdminSlug}, tugas(it, "anggota")))
	id := f.create(pengguna("gotest.au.r6@local.test", nil, []string{"marketing_member"}, tugas(seo, "anggota")))

	// Ganti role yang bertentangan dengan penugasan ditolak, juga lewat jalur RBAC.
	if err := f.repo.Update(f.ctx, id, pengguna("gotest.au.r6@local.test", nil, []string{"finance"}, tugas(seo, "anggota"))); !errors.Is(err, rbac.ErrUnitMismatch) {
		t.Errorf("update: err = %v, harap ErrUnitMismatch", err)
	}
	if err := rbac.NewRepository(f.db).SetUserRoles(f.ctx, int64(id), []string{"finance"}); !errors.Is(err, rbac.ErrUnitMismatch) {
		t.Errorf("SetUserRoles: err = %v, harap ErrUnitMismatch", err)
	}
	if err := rbac.NewRepository(f.db).SetUserRoles(f.ctx, int64(id), []string{"marketing_lead"}); err != nil {
		t.Errorf("SetUserRoles se-divisi: %v", err)
	}
}
