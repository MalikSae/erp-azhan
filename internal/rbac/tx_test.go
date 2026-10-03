package rbac

// Integration test aturan role bersama (tx.go). Semua berjalan di dalam
// transaksi testdb yang selalu di-rollback. Jalan hanya dengan ERP_TEST_DB=1.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// txSerial = testdb.Tx yang diserialkan dengan test pengguna/unit di paket
// lain (lock dipegang sampai transaksi di-rollback).
func txSerial(t *testing.T) *sql.Tx {
	t.Helper()
	testdb.Serialize(t, testdb.DB(t), "erp_test_pengguna")
	_, tx := testdb.Tx(t)
	return tx
}

func TestIntegrasiT5SuperAdminTerakhir(t *testing.T) {
	tx := txSerial(t)
	ctx := context.Background()

	var sisa int64
	if err := tx.QueryRowContext(ctx, `
		SELECT u.id FROM admin_users u JOIN user_roles ur ON ur.admin_user_id = u.id
		JOIN roles r ON r.id = ur.role_id WHERE r.slug = ? ORDER BY u.id LIMIT 1`, SuperAdminSlug).Scan(&sisa); err != nil {
		t.Skipf("belum ada Super Admin di database: %v", err)
	}
	// Sisakan tepat satu Super Admin aktif (di-rollback setelah test).
	testdb.Exec(t, tx, `
		UPDATE admin_users u JOIN user_roles ur ON ur.admin_user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		SET u.is_active = (u.id = ?) WHERE r.slug = ?`, sisa, SuperAdminSlug)

	if err := EnsureOtherActiveSuperAdmin(ctx, tx, sisa); !errors.Is(err, ErrLastSuperAdmin) {
		t.Fatalf("mencabut Super Admin terakhir: err = %v, harap ErrLastSuperAdmin", err)
	}
	if err := EnsureOtherActiveSuperAdmin(ctx, tx, 0); err != nil {
		t.Fatalf("masih ada Super Admin lain: %v", err)
	}
}

func TestIntegrasiValidateRolesScope(t *testing.T) {
	tx := txSerial(t)
	ctx := context.Background()
	brand := int64(1)
	cases := []struct {
		nama  string
		slugs []string
		brand *int64
		want  error
	}{
		{"holding pegang role holding", []string{"direktur"}, nil, nil},
		{"brand pegang role brand", []string{"cs"}, &brand, nil},
		{"role semua di brand", []string{"finance"}, &brand, nil},
		{"role semua di holding", []string{"finance"}, nil, nil},
		{"role holding di brand", []string{"direktur"}, &brand, ErrScopeMismatch},
		{"role brand di holding", []string{"cs"}, nil, ErrScopeMismatch},
		{"tanpa role", nil, nil, ErrNoRole},
		{"role tak dikenal", []string{"tidak_ada"}, nil, ErrUnknownRole},
	}
	for _, c := range cases {
		if err := ValidateRolesScope(ctx, tx, c.slugs, c.brand); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, harap %v", c.nama, err, c.want)
		}
	}
}

func TestIntegrasiValidateUnitsForRoles(t *testing.T) {
	tx := txSerial(t)
	ctx := context.Background()
	unit := func(nama string) int64 {
		var id int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM org_unit WHERE nama = ?`, nama).Scan(&id); err != nil {
			t.Fatalf("unit %q: %v", nama, err)
		}
		return id
	}
	mg, seo, ku, it := unit("Marketing & Growth"), unit("SEO"), unit("Keuangan"), unit("IT")
	brand := int64(1)
	cases := []struct {
		nama  string
		slugs []string
		brand *int64
		units []int64
		ok    bool
	}{
		{"tanpa penugasan selalu sah", []string{"cs"}, &brand, nil, true},
		{"marketing di tim divisinya", []string{"marketing_member"}, nil, []int64{seo}, true},
		{"CMO di divisi M&G", []string{"cmo"}, nil, []int64{mg}, true},
		{"gabungan role menggabung divisi", []string{"cmo", "finance"}, nil, []int64{mg, ku}, true},
		{"Super Admin di mana pun", []string{SuperAdminSlug}, nil, []int64{it, ku}, true},
		{"Finance di Marketing", []string{"finance"}, nil, []int64{mg}, false},
		{"Finance di tim Marketing", []string{"finance"}, nil, []int64{seo}, false},
		{"akses Brand berpenugasan", []string{"finance"}, &brand, []int64{ku}, false},
		{"unit tak dikenal", []string{"finance"}, nil, []int64{-1}, false},
	}
	for _, c := range cases {
		err := ValidateUnitsForRoles(ctx, tx, c.slugs, c.brand, c.units)
		if c.ok && err != nil {
			t.Errorf("%s: err = %v, harap sah", c.nama, err)
		}
		if !c.ok && !errors.Is(err, ErrUnitMismatch) {
			t.Errorf("%s: err = %v, harap ErrUnitMismatch", c.nama, err)
		}
	}
}
