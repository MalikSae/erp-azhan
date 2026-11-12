package rbac

import (
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Temuan audit (INFO): kelola user/role (adminuser, SetUserRoles) tidak memfilter brand, jadi user.admin
// hanya boleh ada di role holding. Route-nya juga dijaga identity.RequireHoldingPermission.
func TestUserAdminHanyaDiRoleHolding(t *testing.T) {
	_, tx := testdb.Tx(t)
	rows, err := tx.Query(`
		SELECT r.slug, r.scope FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE p.code = 'user.admin' AND r.scope <> 'holding'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var slug, scope string
		if err := rows.Scan(&slug, &scope); err != nil {
			t.Fatal(err)
		}
		t.Errorf("role %q (scope %s) memegang user.admin; izin ini hanya untuk role holding", slug, scope)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
