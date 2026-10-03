package orgunit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound       = errors.New("unit tidak ditemukan")
	ErrInvalidJenis   = errors.New("jenis unit harus divisi atau tim")
	ErrInvalidParent  = errors.New("tim wajib berada di bawah divisi; divisi tidak boleh punya induk")
	ErrParentInactive = errors.New("divisi induk sedang nonaktif")
	ErrDuplicateNama  = errors.New("nama unit sudah dipakai di tingkat yang sama")
	ErrHasChildren    = errors.New("unit masih punya tim di bawahnya")
	ErrChildActive    = errors.New("nonaktifkan dulu tim di bawah divisi ini")
	ErrHasMembers     = errors.New("unit masih punya anggota — nonaktifkan saja, jangan dihapus")
	ErrIsRoleDivisi   = errors.New("divisi ini adalah divisi asal sebuah role — tidak bisa dihapus")
	ErrMoveHasMembers = errors.New("tim masih punya anggota — tim tidak bisa dipindah ke divisi lain")
)

// Repository struktur organisasi: pohon 2 tingkat (divisi -> tim).
// Aturan R8 (bentuk pohon) & R9 (unit berisi tidak bisa dihapus) ditegakkan
// di sini, di dalam transaksi.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// ─── Baca ─────────────────────────────────────────────────────────────────────

// Tree mengembalikan divisi beserta tim-nya, jumlah anggota aktif, dan daftar
// pemimpin aktif. Unit nonaktif ikut dikembalikan (UI memberi tanda).
func (r *Repository) Tree(ctx context.Context) ([]Unit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, parent_id, jenis, nama, urutan, is_active
		FROM org_unit ORDER BY urutan, nama`)
	if err != nil {
		return nil, fmt.Errorf("orgunit.Tree: %w", err)
	}
	defer rows.Close()

	all := map[uint64]*Unit{}
	order := []uint64{}
	for rows.Next() {
		var u Unit
		var parent sql.NullInt64
		if err := rows.Scan(&u.ID, &parent, &u.Jenis, &u.Nama, &u.Urutan, &u.IsActive); err != nil {
			return nil, fmt.Errorf("orgunit.Tree scan: %w", err)
		}
		if parent.Valid {
			p := uint64(parent.Int64)
			u.ParentID = &p
		}
		u.Pemimpin = []Pemimpin{}
		if u.Jenis == "divisi" {
			u.Role = []string{}
		}
		all[u.ID] = &u
		order = append(order, u.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Anggota & pemimpin: hanya pengguna aktif (yang nonaktif sudah offboard).
	mRows, err := r.db.QueryContext(ctx, `
		SELECT m.org_unit_id, m.posisi, u.id, COALESCE(u.display_name, u.email)
		FROM org_member m JOIN admin_users u ON u.id = m.admin_user_id
		WHERE u.is_active = TRUE
		ORDER BY u.display_name`)
	if err != nil {
		return nil, fmt.Errorf("orgunit.Tree anggota: %w", err)
	}
	defer mRows.Close()
	for mRows.Next() {
		var unitID, userID uint64
		var posisi, nama string
		if err := mRows.Scan(&unitID, &posisi, &userID, &nama); err != nil {
			return nil, err
		}
		u, ok := all[unitID]
		if !ok {
			continue
		}
		u.JumlahAnggota++
		if posisi == "pemimpin" {
			u.Pemimpin = append(u.Pemimpin, Pemimpin{ID: userID, DisplayName: nama})
		}
	}
	if err := mRows.Err(); err != nil {
		return nil, err
	}

	// Role yang berdivisi asal di sini (R10) — informasi untuk admin.
	rRows, err := r.db.QueryContext(ctx,
		`SELECT divisi_id, name FROM roles WHERE divisi_id IS NOT NULL ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("orgunit.Tree role: %w", err)
	}
	defer rRows.Close()
	for rRows.Next() {
		var unitID uint64
		var name string
		if err := rRows.Scan(&unitID, &name); err != nil {
			return nil, err
		}
		if u, ok := all[unitID]; ok {
			u.Role = append(u.Role, name)
		}
	}
	if err := rRows.Err(); err != nil {
		return nil, err
	}

	tree := []Unit{}
	for _, id := range order {
		u := all[id]
		if u.Jenis == "divisi" {
			tim := []Unit{}
			for _, cid := range order {
				c := all[cid]
				if c.ParentID != nil && *c.ParentID == u.ID {
					tim = append(tim, *c)
				}
			}
			u.Tim = &tim
			tree = append(tree, *u)
		}
	}
	return tree, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*Unit, error) {
	var u Unit
	var parent sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, parent_id, jenis, nama, urutan, is_active FROM org_unit WHERE id = ?`, id).
		Scan(&u.ID, &parent, &u.Jenis, &u.Nama, &u.Urutan, &u.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("orgunit.GetByID: %w", err)
	}
	if parent.Valid {
		p := uint64(parent.Int64)
		u.ParentID = &p
	}
	u.Pemimpin = []Pemimpin{}
	return &u, nil
}

// ─── Helper aturan (di dalam transaksi) ──────────────────────────────────────

// validateParent menegakkan R8: divisi tanpa induk; tim wajib di bawah
// divisi AKTIF (tim di bawah tim tidak diizinkan → maksimal 2 tingkat).
func validateParent(ctx context.Context, tx *sql.Tx, jenis string, parentID *uint64) error {
	switch jenis {
	case "divisi":
		if parentID != nil {
			return ErrInvalidParent
		}
		return nil
	case "tim":
		if parentID == nil {
			return ErrInvalidParent
		}
		var pJenis string
		var pActive bool
		err := tx.QueryRowContext(ctx,
			`SELECT jenis, is_active FROM org_unit WHERE id = ? FOR UPDATE`, *parentID).Scan(&pJenis, &pActive)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && pJenis != "divisi") {
			return ErrInvalidParent
		}
		if err != nil {
			return fmt.Errorf("orgunit.validateParent: %w", err)
		}
		if !pActive {
			return ErrParentInactive
		}
		return nil
	default:
		return ErrInvalidJenis
	}
}

// checkNamaUnik: nama unik di antara saudara (induk yang sama). Dicek di
// aplikasi karena UNIQUE MySQL tidak menganggap NULL sama.
func checkNamaUnik(ctx context.Context, tx *sql.Tx, nama string, parentID *uint64, excludeID uint64) error {
	var n int
	var err error
	if parentID == nil {
		err = tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM org_unit WHERE parent_id IS NULL AND nama = ? AND id != ?`,
			nama, excludeID).Scan(&n)
	} else {
		err = tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM org_unit WHERE parent_id = ? AND nama = ? AND id != ?`,
			*parentID, nama, excludeID).Scan(&n)
	}
	if err != nil {
		return fmt.Errorf("orgunit.checkNamaUnik: %w", err)
	}
	if n > 0 {
		return ErrDuplicateNama
	}
	return nil
}

// ─── Tulis ────────────────────────────────────────────────────────────────────

func (r *Repository) Create(ctx context.Context, in CreateRequest) (uint64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("orgunit.Create begin: %w", err)
	}
	defer tx.Rollback()

	if err := validateParent(ctx, tx, in.Jenis, in.ParentID); err != nil {
		return 0, err
	}
	if err := checkNamaUnik(ctx, tx, in.Nama, in.ParentID, 0); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO org_unit (parent_id, jenis, nama, urutan, is_active) VALUES (?, ?, ?, ?, TRUE)`,
		in.ParentID, in.Jenis, in.Nama, in.Urutan)
	if err != nil {
		return 0, fmt.Errorf("orgunit.Create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("orgunit.Create id: %w", err)
	}
	return uint64(id), tx.Commit()
}

// Update mengubah nama/urutan; tim boleh dipindah ke divisi lain. Jenis
// tidak bisa diubah (mengubah divisi jadi tim akan merusak pohon).
func (r *Repository) Update(ctx context.Context, id uint64, in UpdateRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("orgunit.Update begin: %w", err)
	}
	defer tx.Rollback()

	var jenis string
	var oldParent sql.NullInt64
	err = tx.QueryRowContext(ctx,
		`SELECT jenis, parent_id FROM org_unit WHERE id = ? FOR UPDATE`, id).Scan(&jenis, &oldParent)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("orgunit.Update lock: %w", err)
	}

	// Pindah induk hanya divalidasi bila memang berubah, supaya tim di bawah
	// divisi nonaktif masih bisa diganti namanya.
	parentChanged := oldParent.Valid != (in.ParentID != nil) ||
		(oldParent.Valid && in.ParentID != nil && uint64(oldParent.Int64) != *in.ParentID)
	if parentChanged {
		if err := validateParent(ctx, tx, jenis, in.ParentID); err != nil {
			return err
		}
		// R10: anggota tim terikat divisi asal role-nya; memindah tim berisi
		// diam-diam membuat penugasan mereka tidak sah.
		var members int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM org_member WHERE org_unit_id = ?`, id).Scan(&members); err != nil {
			return fmt.Errorf("orgunit.Update cek anggota: %w", err)
		}
		if members > 0 {
			return ErrMoveHasMembers
		}
	} else if (jenis == "divisi") != (in.ParentID == nil) {
		return ErrInvalidParent
	}
	if err := checkNamaUnik(ctx, tx, in.Nama, in.ParentID, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE org_unit SET nama = ?, parent_id = ?, urutan = ? WHERE id = ?`,
		in.Nama, in.ParentID, in.Urutan, id); err != nil {
		return fmt.Errorf("orgunit.Update: %w", err)
	}
	return tx.Commit()
}

// SetStatus mengaktifkan/menonaktifkan unit. Tanpa efek samping tersembunyi:
// divisi tidak bisa dinonaktifkan selama masih ada tim aktif di bawahnya, dan
// tim tidak bisa diaktifkan bila divisi induknya nonaktif. Keanggotaan lama
// TIDAK dihapus (riwayat penugasan tetap utuh).
func (r *Repository) SetStatus(ctx context.Context, id uint64, active bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("orgunit.SetStatus begin: %w", err)
	}
	defer tx.Rollback()

	var jenis string
	var parent sql.NullInt64
	var isActive bool
	err = tx.QueryRowContext(ctx,
		`SELECT jenis, parent_id, is_active FROM org_unit WHERE id = ? FOR UPDATE`, id).
		Scan(&jenis, &parent, &isActive)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("orgunit.SetStatus lock: %w", err)
	}
	if isActive == active {
		return tx.Commit()
	}

	if !active && jenis == "divisi" {
		var n int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM org_unit WHERE parent_id = ? AND is_active = TRUE`, id).Scan(&n); err != nil {
			return fmt.Errorf("orgunit.SetStatus cek tim: %w", err)
		}
		if n > 0 {
			return ErrChildActive
		}
	}
	if active && jenis == "tim" && parent.Valid {
		var pActive bool
		if err := tx.QueryRowContext(ctx,
			`SELECT is_active FROM org_unit WHERE id = ? FOR UPDATE`, parent.Int64).Scan(&pActive); err != nil {
			return fmt.Errorf("orgunit.SetStatus cek induk: %w", err)
		}
		if !pActive {
			return ErrParentInactive
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE org_unit SET is_active = ? WHERE id = ?`, active, id); err != nil {
		return fmt.Errorf("orgunit.SetStatus: %w", err)
	}
	return tx.Commit()
}

// Delete menghapus permanen HANYA unit kosong (tanpa tim & tanpa anggota,
// termasuk anggota nonaktif) — R9. Unit berisi: nonaktifkan.
func (r *Repository) Delete(ctx context.Context, id uint64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("orgunit.Delete begin: %w", err)
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM org_unit WHERE id = ? FOR UPDATE`, id).Scan(&exists); err != nil {
		return fmt.Errorf("orgunit.Delete lock: %w", err)
	}
	if exists == 0 {
		return ErrNotFound
	}
	var children, members int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM org_unit WHERE parent_id = ?`, id).Scan(&children); err != nil {
		return fmt.Errorf("orgunit.Delete cek tim: %w", err)
	}
	if children > 0 {
		return ErrHasChildren
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM org_member WHERE org_unit_id = ?`, id).Scan(&members); err != nil {
		return fmt.Errorf("orgunit.Delete cek anggota: %w", err)
	}
	if members > 0 {
		return ErrHasMembers
	}
	var roleRefs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM roles WHERE divisi_id = ?`, id).Scan(&roleRefs); err != nil {
		return fmt.Errorf("orgunit.Delete cek role: %w", err)
	}
	if roleRefs > 0 {
		return ErrIsRoleDivisi
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM org_unit WHERE id = ?`, id); err != nil {
		// Jaring pengaman: FK RESTRICT bila ada baris yang lolos pengecekan.
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1451 {
			return ErrHasMembers
		}
		return fmt.Errorf("orgunit.Delete: %w", err)
	}
	return tx.Commit()
}
