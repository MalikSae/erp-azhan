package benefit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound = errors.New("benefit tidak ditemukan")
	// ErrDuplikat: teks yang sama (tanpa membedakan huruf besar-kecil) sudah ada di brand itu (409).
	ErrDuplikat = errors.New("benefit dengan teks yang sama sudah ada di brand ini")
	// ErrDuplikatNonaktif: teks yang sama sudah ada tetapi nonaktif, sehingga tidak muncul di pilihan paket (409).
	ErrDuplikatNonaktif = errors.New("benefit dengan teks yang sama sudah ada tetapi nonaktif; aktifkan lagi di menu Benefit Paket")
	// ErrDipakai: benefit masih dipakai paket sehingga tidak bisa dihapus (409).
	ErrDipakai = errors.New("benefit masih dipakai paket; nonaktifkan saja")
	// ErrBrandTidakSah: brand_id yang dikirim admin holding tidak ada (400).
	ErrBrandTidakSah = errors.New("brand_id tidak valid")
)

// Querier dipenuhi *sql.DB dan *sql.Tx, supaya paket schedule bisa memakai fungsi di sini di dalam
// transaksinya sendiri.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectBenefit = `
	SELECT b.id, b.brand_id, b.teks, b.ikon, b.is_active,
		(SELECT COUNT(*) FROM schedule_benefits sb WHERE sb.benefit_id = b.id) AS dipakai,
		b.created_at, b.updated_at
	FROM paket_benefits b`

func scanBenefit(s interface{ Scan(dest ...any) error }) (Benefit, error) {
	var b Benefit
	err := s.Scan(&b.ID, &b.BrandID, &b.Teks, &b.Ikon, &b.IsActive, &b.Dipakai, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

// List mengambil benefit satu brand terurut teks, beserta jumlah paket yang memakainya. hanyaAktif
// membatasi ke benefit yang masih ditawarkan untuk pilihan baru.
func (r *Repository) List(ctx context.Context, brandID int64, hanyaAktif bool) ([]Benefit, error) {
	return list(ctx, r.db, brandID, hanyaAktif)
}

func list(ctx context.Context, q Querier, brandID int64, hanyaAktif bool) ([]Benefit, error) {
	query := selectBenefit + " WHERE b.brand_id = ?"
	if hanyaAktif {
		query += " AND b.is_active = 1"
	}
	query += " ORDER BY b.teks ASC, b.id ASC"
	rows, err := q.QueryContext(ctx, query, brandID)
	if err != nil {
		return nil, fmt.Errorf("benefit.List: %w", err)
	}
	defer rows.Close()
	items := make([]Benefit, 0)
	for rows.Next() {
		b, err := scanBenefit(rows)
		if err != nil {
			return nil, fmt.Errorf("benefit.List scan: %w", err)
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

// GetByID mengambil satu benefit. scope non-nil membatasi ke brand itu (ErrNotFound bila bukan miliknya).
func (r *Repository) GetByID(ctx context.Context, id int64, scope *int64) (*Benefit, error) {
	return getByID(ctx, r.db, id, scope)
}

func getByID(ctx context.Context, q Querier, id int64, scope *int64) (*Benefit, error) {
	b, err := scanBenefit(q.QueryRowContext(ctx, selectBenefit+" WHERE b.id = ? AND (? IS NULL OR b.brand_id = ?)", id, scope, scope))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("benefit.GetByID: %w", err)
	}
	return &b, nil
}

// galatTulis menerjemahkan error MySQL saat menulis benefit: 1062 teks kembar, 1452 brand tidak ada.
func galatTulis(err error) error {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		switch me.Number {
		case 1062:
			return ErrDuplikat
		case 1452:
			return ErrBrandTidakSah
		}
	}
	return err
}

// kembarNonaktif melaporkan apakah benefit berteks sama di brand itu sedang nonaktif. Admin paket tidak
// melihat benefit nonaktif di pilihannya, jadi pesan galatnya harus menyebut sebabnya.
func kembarNonaktif(ctx context.Context, q Querier, brandID int64, teks string) bool {
	var aktif bool
	err := q.QueryRowContext(ctx, `SELECT is_active FROM paket_benefits WHERE brand_id = ? AND teks = ?`, brandID, teks).Scan(&aktif)
	return err == nil && !aktif
}

// Create menyisipkan benefit baru (aktif). teks dan ikon sudah divalidasi pemanggil (ValidasiIsi).
func (r *Repository) Create(ctx context.Context, brandID int64, teks, ikon string) (*Benefit, error) {
	return create(ctx, r.db, brandID, teks, ikon)
}

func create(ctx context.Context, q Querier, brandID int64, teks, ikon string) (*Benefit, error) {
	res, err := q.ExecContext(ctx, `INSERT INTO paket_benefits (brand_id, teks, ikon) VALUES (?, ?, ?)`, brandID, teks, ikon)
	if err != nil {
		if e := galatTulis(err); e != err {
			if errors.Is(e, ErrDuplikat) && kembarNonaktif(ctx, q, brandID, teks) {
				return nil, ErrDuplikatNonaktif
			}
			return nil, e
		}
		return nil, fmt.Errorf("benefit.Create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("benefit.Create lastInsertId: %w", err)
	}
	return getByID(ctx, q, id, nil)
}

// Update mengubah teks, ikon, dan (bila isActive tidak nil) status aktif. Perubahan berlaku untuk semua
// paket yang memakai benefit ini karena paket menautkannya lewat id.
func (r *Repository) Update(ctx context.Context, id int64, scope *int64, teks, ikon string, isActive *bool) (*Benefit, error) {
	return update(ctx, r.db, id, scope, teks, ikon, isActive)
}

func update(ctx context.Context, q Querier, id int64, scope *int64, teks, ikon string, isActive *bool) (*Benefit, error) {
	// Dicek dulu: UPDATE tanpa perubahan nilai melaporkan 0 baris, sama seperti id yang tidak ada.
	if _, err := getByID(ctx, q, id, scope); err != nil {
		return nil, err
	}
	ubahAktif := isActive != nil
	aktif := false
	if ubahAktif {
		aktif = *isActive
	}
	if _, err := q.ExecContext(ctx,
		`UPDATE paket_benefits SET teks = ?, ikon = ?, is_active = IF(?, ?, is_active) WHERE id = ?`,
		teks, ikon, ubahAktif, aktif, id); err != nil {
		if e := galatTulis(err); e != err {
			return nil, e
		}
		return nil, fmt.Errorf("benefit.Update: %w", err)
	}
	return getByID(ctx, q, id, scope)
}

// Delete menghapus benefit yang tidak dipakai paket mana pun.
func (r *Repository) Delete(ctx context.Context, id int64, scope *int64) error {
	return hapus(ctx, r.db, id, scope)
}

func hapus(ctx context.Context, q Querier, id int64, scope *int64) error {
	b, err := getByID(ctx, q, id, scope)
	if err != nil {
		return err
	}
	if b.Dipakai > 0 {
		return ErrDipakai
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM paket_benefits WHERE id = ?`, id); err != nil {
		// 1451: paket menautkan benefit ini di antara pemeriksaan di atas dan DELETE (FK RESTRICT).
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1451 {
			return ErrDipakai
		}
		return fmt.Errorf("benefit.Delete: %w", err)
	}
	return nil
}

// ─── Dipakai paket schedule ──────────────────────────────────────────────────

func placeholder(ids []int64) (string, []any) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return "?" + strings.Repeat(",?", len(ids)-1), args
}

// ValidasiPilihan memeriksa daftar benefit yang dipilih sebuah paket: bentuk daftar (ValidasiDaftarID),
// setiap benefit milik brand paket, dan benefit nonaktif hanya diterima bila paket itu sudah memakainya.
// scheduleID 0 untuk paket baru. Daftar yang tidak sah dikembalikan sebagai PilihanError.
func ValidasiPilihan(ctx context.Context, q Querier, brandID, scheduleID int64, ids []int64) error {
	if err := ValidasiDaftarID(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	ph, args := placeholder(ids)
	args = append([]any{scheduleID, brandID}, args...)
	rows, err := q.QueryContext(ctx, `
		SELECT b.id, b.teks, b.is_active,
			EXISTS (SELECT 1 FROM schedule_benefits sb WHERE sb.schedule_id = ? AND sb.benefit_id = b.id)
		FROM paket_benefits b
		WHERE b.brand_id = ? AND b.id IN (`+ph+`)`, args...)
	if err != nil {
		return fmt.Errorf("benefit.ValidasiPilihan: %w", err)
	}
	defer rows.Close()
	ada := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		var teks string
		var aktif, sudahDipakai bool
		if err := rows.Scan(&id, &teks, &aktif, &sudahDipakai); err != nil {
			return fmt.Errorf("benefit.ValidasiPilihan scan: %w", err)
		}
		if !aktif && !sudahDipakai {
			return PilihanError(fmt.Sprintf("benefit %q sudah nonaktif, pilih benefit lain", teks))
		}
		ada[id] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("benefit.ValidasiPilihan: %w", err)
	}
	for _, id := range ids {
		if !ada[id] {
			return PilihanError("highlight benefit tidak ditemukan di brand paket ini")
		}
	}
	return nil
}

// GantiTautan mengganti seluruh benefit sebuah paket (hapus lalu sisip) dengan urutan sesuai daftar.
// Dipanggil di dalam transaksi simpan paket; daftar sudah lolos ValidasiPilihan.
func GantiTautan(ctx context.Context, tx Querier, scheduleID int64, ids []int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM schedule_benefits WHERE schedule_id = ?`, scheduleID); err != nil {
		return fmt.Errorf("benefit.GantiTautan delete: %w", err)
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schedule_benefits (schedule_id, benefit_id, urutan) VALUES (?, ?, ?)`, scheduleID, id, i+1); err != nil {
			return fmt.Errorf("benefit.GantiTautan insert: %w", err)
		}
	}
	return nil
}

// UntukJadwal mengambil benefit beberapa paket sekaligus, terurut per paket. Benefit nonaktif tetap ikut.
// Paket tanpa benefit tidak punya kunci di peta hasil.
func UntukJadwal(ctx context.Context, q Querier, scheduleIDs []int64) (map[int64][]Ref, error) {
	out := make(map[int64][]Ref, len(scheduleIDs))
	if len(scheduleIDs) == 0 {
		return out, nil
	}
	ph, args := placeholder(scheduleIDs)
	rows, err := q.QueryContext(ctx, `
		SELECT sb.schedule_id, b.id, b.teks, b.ikon, b.is_active
		FROM schedule_benefits sb
		JOIN paket_benefits b ON b.id = sb.benefit_id
		WHERE sb.schedule_id IN (`+ph+`)
		ORDER BY sb.schedule_id, sb.urutan ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("benefit.UntukJadwal: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sid int64
		var ref Ref
		if err := rows.Scan(&sid, &ref.ID, &ref.Teks, &ref.Ikon, &ref.IsActive); err != nil {
			return nil, fmt.Errorf("benefit.UntukJadwal scan: %w", err)
		}
		out[sid] = append(out[sid], ref)
	}
	return out, rows.Err()
}

// JumlahTautan menghitung benefit yang tersimpan pada sebuah paket.
func JumlahTautan(ctx context.Context, q Querier, scheduleID int64) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM schedule_benefits WHERE schedule_id = ?`, scheduleID).Scan(&n)
	return n, err
}
