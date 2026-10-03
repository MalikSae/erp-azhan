package orgunit

// Integration test struktur organisasi (SPEK-PENGGUNA-ORGANISASI R8–R10,
// T10–T11). Repository membuka transaksinya sendiri, jadi unit uji (nama
// "GoTest …") dibuat sungguhan lalu dibersihkan. Jalan hanya dengan
// ERP_TEST_DB=1.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

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
		for _, q := range []string{
			`DELETE FROM admin_users WHERE email LIKE 'gotest.ou.%@local.test'`,
			`DELETE FROM roles WHERE slug LIKE 'gotest_%'`,
			`DELETE FROM org_unit WHERE jenis = 'tim' AND nama LIKE 'GoTest%'`,
			`DELETE FROM org_unit WHERE nama LIKE 'GoTest%'`,
		} {
			if _, err := db.Exec(q); err != nil {
				t.Errorf("bersihkan %q: %v", q, err)
			}
		}
	}
	clean()
	t.Cleanup(clean)
	return &fixture{t: t, db: db, repo: NewRepository(db), ctx: context.Background()}
}

func (f *fixture) divisi(nama string) uint64 {
	f.t.Helper()
	id, err := f.repo.Create(f.ctx, CreateRequest{Jenis: "divisi", Nama: nama})
	if err != nil {
		f.t.Fatalf("buat divisi %q: %v", nama, err)
	}
	return id
}

func (f *fixture) tim(nama string, divisi uint64) uint64 {
	f.t.Helper()
	id, err := f.repo.Create(f.ctx, CreateRequest{Jenis: "tim", Nama: nama, ParentID: &divisi})
	if err != nil {
		f.t.Fatalf("buat tim %q: %v", nama, err)
	}
	return id
}

// anggota menaruh pengguna uji langsung di unit (lewat SQL: aturan penugasan
// pengguna diuji di paket adminuser; di sini yang diuji aturan unit).
func (f *fixture) anggota(unitID uint64) {
	f.t.Helper()
	res, err := f.db.Exec(`INSERT INTO admin_users (email, password_hash) VALUES ('gotest.ou.org@local.test', 'x')`)
	if err != nil {
		f.t.Fatal(err)
	}
	uid, _ := res.LastInsertId()
	if _, err := f.db.Exec(`INSERT INTO org_member (admin_user_id, org_unit_id, posisi) VALUES (?, ?, 'anggota')`, uid, unitID); err != nil {
		f.t.Fatal(err)
	}
}

func TestIntegrasiT11BentukPohon(t *testing.T) {
	f := setup(t)
	d := f.divisi("GoTest Divisi")
	tm := f.tim("GoTest Tim", d)

	cases := []struct {
		nama string
		in   CreateRequest
	}{
		{"tim tanpa divisi", CreateRequest{Jenis: "tim", Nama: "GoTest Yatim"}},
		{"tim di bawah tim", CreateRequest{Jenis: "tim", Nama: "GoTest Cucu", ParentID: &tm}},
		{"divisi berinduk", CreateRequest{Jenis: "divisi", Nama: "GoTest Anak", ParentID: &d}},
	}
	for _, c := range cases {
		if _, err := f.repo.Create(f.ctx, c.in); !errors.Is(err, ErrInvalidParent) {
			t.Errorf("%s: err = %v, harap ErrInvalidParent", c.nama, err)
		}
	}
	if _, err := f.repo.Create(f.ctx, CreateRequest{Jenis: "tim", Nama: "GoTest Tim", ParentID: &d}); !errors.Is(err, ErrDuplicateNama) {
		t.Errorf("nama dobel: err = %v, harap ErrDuplicateNama", err)
	}
}

func TestIntegrasiT10UnitBerisiTidakBisaDihapus(t *testing.T) {
	f := setup(t)
	d := f.divisi("GoTest Divisi")
	tm := f.tim("GoTest Tim", d)
	f.anggota(tm)

	if err := f.repo.Delete(f.ctx, d); !errors.Is(err, ErrHasChildren) {
		t.Errorf("hapus divisi bertim: err = %v, harap ErrHasChildren", err)
	}
	if err := f.repo.Delete(f.ctx, tm); !errors.Is(err, ErrHasMembers) {
		t.Errorf("hapus tim beranggota: err = %v, harap ErrHasMembers", err)
	}
	if err := f.repo.SetStatus(f.ctx, d, false); !errors.Is(err, ErrChildActive) {
		t.Errorf("nonaktifkan divisi bertim aktif: err = %v, harap ErrChildActive", err)
	}
	if err := f.repo.SetStatus(f.ctx, tm, false); err != nil {
		t.Fatalf("nonaktifkan tim: %v", err)
	}
	if err := f.repo.SetStatus(f.ctx, d, false); err != nil {
		t.Fatalf("nonaktifkan divisi: %v", err)
	}
	if err := f.repo.SetStatus(f.ctx, tm, true); !errors.Is(err, ErrParentInactive) {
		t.Errorf("aktifkan tim di divisi nonaktif: err = %v, harap ErrParentInactive", err)
	}
}

func TestIntegrasiR10UnitTerikatRole(t *testing.T) {
	f := setup(t)
	d1, d2 := f.divisi("GoTest Divisi A"), f.divisi("GoTest Divisi B")
	tm := f.tim("GoTest Tim", d1)

	// Divisi asal sebuah role tidak bisa dihapus.
	if _, err := f.db.Exec(`INSERT INTO roles (slug, name, scope, divisi_id) VALUES ('gotest_role', 'GoTest Role', 'holding', ?)`, d2); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.Delete(f.ctx, d2); !errors.Is(err, ErrIsRoleDivisi) {
		t.Errorf("hapus divisi asal role: err = %v, harap ErrIsRoleDivisi", err)
	}

	// Tim kosong boleh pindah divisi; tim berisi tidak.
	if err := f.repo.Update(f.ctx, tm, UpdateRequest{Nama: "GoTest Tim", ParentID: &d2}); err != nil {
		t.Fatalf("pindah tim kosong: %v", err)
	}
	f.anggota(tm)
	if err := f.repo.Update(f.ctx, tm, UpdateRequest{Nama: "GoTest Tim", ParentID: &d1}); !errors.Is(err, ErrMoveHasMembers) {
		t.Errorf("pindah tim berisi: err = %v, harap ErrMoveHasMembers", err)
	}
}
