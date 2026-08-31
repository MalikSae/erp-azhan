package jamaah

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Sentinel errors
var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrDuplicateNIK    = errors.New("NIK sudah terdaftar")
	ErrKodeBrandNotSet = errors.New("kode_brand belum diatur untuk brand ini, hubungi Super Admin untuk mengatur di Kelola Brand")
)

const kodeJamaahCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Repository mengelola semua query ke tabel jamaah.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) generateUniqueKodeJamaahTx(ctx context.Context, tx *sql.Tx) (string, error) {
	charsetLen := big.NewInt(int64(len(kodeJamaahCharset)))
	for {
		b := make([]byte, 6)
		for i := 0; i < 6; i++ {
			num, err := rand.Int(rand.Reader, charsetLen)
			if err != nil {
				return "", fmt.Errorf("generate kode jamaah: %w", err)
			}
			b[i] = kodeJamaahCharset[num.Int64()]
		}
		code := string(b)

		var count int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM jamaah WHERE kode_jamaah = ?", code).Scan(&count)
		if err != nil {
			return "", fmt.Errorf("check kode jamaah collision: %w", err)
		}
		if count == 0 {
			return code, nil
		}
	}
}

// GenerateIDJamaah creates formatted ID {KodeBrand}-{YYMM}{Counter6digit} with row lock on brands table.
func (r *Repository) GenerateIDJamaah(ctx context.Context, tx *sql.Tx, brandID int64) (string, error) {
	var (
		kodeBrand sql.NullString
		counter   uint32
	)

	// 1. SELECT kode_brand, jamaah_counter FROM brands WHERE id = ? FOR UPDATE
	q := "SELECT kode_brand, jamaah_counter FROM brands WHERE id = ? FOR UPDATE"
	err := tx.QueryRowContext(ctx, q, brandID).Scan(&kodeBrand, &counter)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("lock brand: %w", err)
	}

	// 2. Validate kode_brand is set
	if !kodeBrand.Valid || strings.TrimSpace(kodeBrand.String) == "" {
		return "", ErrKodeBrandNotSet
	}

	// 3. Increment counter
	newCounter := counter + 1

	// 4. UPDATE brands SET jamaah_counter = ? WHERE id = ?
	_, err = tx.ExecContext(ctx, "UPDATE brands SET jamaah_counter = ? WHERE id = ?", newCounter, brandID)
	if err != nil {
		return "", fmt.Errorf("update jamaah_counter: %w", err)
	}

	// 5. Format: {kode_brand}-{YY}{MM}{newCounter 6-digit}
	now := time.Now()
	idJamaah := fmt.Sprintf("%s-%02d%02d%06d", strings.ToUpper(strings.TrimSpace(kodeBrand.String)), now.Year()%100, int(now.Month()), newCounter)

	return idJamaah, nil
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List mengambil semua jamaah, filter brand kalau scoped admin.
func (r *Repository) List(ctx context.Context, brandID *int64, status string) ([]JamaahListItem, error) {
	q := `SELECT id, brand_id, COALESCE(id_jamaah, ''), kode_jamaah, nama_lengkap, nik,
		DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') AS tanggal_lahir,
		no_hp, email, status, created_at
		FROM jamaah WHERE 1=1`

	var args []interface{}
	if brandID != nil {
		q += " AND brand_id = ?"
		args = append(args, *brandID)
	}

	if status == "draft" {
		q += " AND status = 'draft'"
	} else if status == "aktif" {
		q += " AND status = 'aktif'"
	} else if status != "" && status != "all" {
		q += " AND status = ?"
		args = append(args, status)
	}

	q += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("jamaah.List: %w", err)
	}
	defer rows.Close()

	items := make([]JamaahListItem, 0)
	for rows.Next() {
		var item JamaahListItem
		if err := rows.Scan(&item.ID, &item.BrandID, &item.IDJamaah, &item.KodeJamaah, &item.NamaLengkap, &item.NIK, &item.TanggalLahir, &item.NoHP, &item.Email, &item.Status, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("jamaah.List scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

// GetByID mengambil jamaah lengkap, verify brand match untuk scoped admin.
func (r *Repository) GetByID(ctx context.Context, id int64, brandID *int64) (*Jamaah, error) {
	q := `SELECT id, brand_id, COALESCE(id_jamaah, ''), kode_jamaah, nama_lengkap, nama_ayah_kandung, nik, tempat_lahir,
		DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') AS tanggal_lahir,
		no_paspor, tempat_paspor_keluar,
		DATE_FORMAT(tanggal_paspor_keluar, '%Y-%m-%d') AS tanggal_paspor_keluar,
		DATE_FORMAT(paspor_berlaku_sampai, '%Y-%m-%d') AS paspor_berlaku_sampai,
		no_hp, email, pekerjaan, pendidikan_terakhir, penjamin_kesehatan, no_asuransi_bpjs,
		alamat, kota, catatan, status, emergency_nama, emergency_nik, emergency_hp, emergency_hubungan, emergency_alamat,
		created_at
		FROM jamaah WHERE id=?`

	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}

	var j Jamaah
	err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&j.ID, &j.BrandID, &j.IDJamaah, &j.KodeJamaah, &j.NamaLengkap, &j.NamaAyahKandung, &j.NIK, &j.TempatLahir,
		&j.TanggalLahir,
		&j.NoPaspor, &j.TempatPasporKeluar, &j.TanggalPasporKeluar,
		&j.PasporBerlakuSampai,
		&j.NoHP, &j.Email, &j.Pekerjaan, &j.PendidikanTerakhir, &j.PenjaminKesehatan, &j.NoAsuransiBPJS,
		&j.Alamat, &j.Kota, &j.Catatan, &j.Status, &j.EmergencyNama, &j.EmergencyNIK, &j.EmergencyHP, &j.EmergencyHubungan, &j.EmergencyAlamat,
		&j.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("jamaah.GetByID: %w", err)
	}
	return &j, nil
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create menyisipkan jamaah baru dengan id_jamaah terstruktur & kode_jamaah. Tangkap duplicate NIK.
func (r *Repository) Create(ctx context.Context, brandID int64, req *CreateJamaahRequest) (*Jamaah, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	idJamaah, err := r.GenerateIDJamaah(ctx, tx, brandID)
	if err != nil {
		return nil, err
	}

	kodeJamaah, err := r.generateUniqueKodeJamaahTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	status := "aktif"
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = strings.TrimSpace(*req.Status)
	}

	const q = `INSERT INTO jamaah (
		brand_id, id_jamaah, kode_jamaah, nama_lengkap, nama_ayah_kandung, nik, tempat_lahir, tanggal_lahir,
		no_paspor, tempat_paspor_keluar, tanggal_paspor_keluar, paspor_berlaku_sampai,
		no_hp, email, pekerjaan, pendidikan_terakhir, penjamin_kesehatan, no_asuransi_bpjs,
		alamat, kota, catatan, status, emergency_nama, emergency_nik, emergency_hp, emergency_hubungan, emergency_alamat
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.ExecContext(ctx, q,
		brandID, idJamaah, kodeJamaah, req.NamaLengkap, req.NamaAyahKandung, req.NIK, req.TempatLahir, req.TanggalLahir,
		req.NoPaspor, req.TempatPasporKeluar, req.TanggalPasporKeluar, req.PasporBerlakuSampai,
		req.NoHP, req.Email, req.Pekerjaan, req.PendidikanTerakhir, req.PenjaminKesehatan, req.NoAsuransiBPJS,
		req.Alamat, req.Kota, req.Catatan, status, req.EmergencyNama, req.EmergencyNIK, req.EmergencyHP, req.EmergencyHubungan, req.EmergencyAlamat,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrDuplicateNIK
		}
		return nil, fmt.Errorf("jamaah.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("jamaah.Create LastInsertId: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetByID(ctx, id, nil)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Update memperbarui data jamaah. Verify brand match dulu.
func (r *Repository) Update(ctx context.Context, id int64, brandID *int64, finalBrandID int64, req *UpdateJamaahRequest) (*Jamaah, error) {
	existing, err := r.GetByID(ctx, id, brandID)
	if err != nil {
		return nil, err
	}

	status := existing.Status
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = strings.TrimSpace(*req.Status)
	}

	const q = `UPDATE jamaah SET
		brand_id=?, nama_lengkap=?, nama_ayah_kandung=?, nik=?, tempat_lahir=?, tanggal_lahir=?,
		no_paspor=?, tempat_paspor_keluar=?, tanggal_paspor_keluar=?, paspor_berlaku_sampai=?,
		no_hp=?, email=?, pekerjaan=?, pendidikan_terakhir=?, penjamin_kesehatan=?, no_asuransi_bpjs=?,
		alamat=?, kota=?, catatan=?, status=?, emergency_nama=?, emergency_nik=?, emergency_hp=?, emergency_hubungan=?, emergency_alamat=?
		WHERE id=?`

	_, err = r.db.ExecContext(ctx, q,
		finalBrandID, req.NamaLengkap, req.NamaAyahKandung, req.NIK, req.TempatLahir, req.TanggalLahir,
		req.NoPaspor, req.TempatPasporKeluar, req.TanggalPasporKeluar, req.PasporBerlakuSampai,
		req.NoHP, req.Email, req.Pekerjaan, req.PendidikanTerakhir, req.PenjaminKesehatan, req.NoAsuransiBPJS,
		req.Alamat, req.Kota, req.Catatan, status, req.EmergencyNama, req.EmergencyNIK, req.EmergencyHP, req.EmergencyHubungan, req.EmergencyAlamat,
		id,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrDuplicateNIK
		}
		return nil, fmt.Errorf("jamaah.Update: %w", err)
	}
	return r.GetByID(ctx, id, brandID)
}

// ─── Update Catatan ───────────────────────────────────────────────────────────

// UpdateCatatan memperbarui field catatan saja.
func (r *Repository) UpdateCatatan(ctx context.Context, id int64, brandID *int64, catatan *string) error {
	if err := r.exists(ctx, id, brandID); err != nil {
		return err
	}
	q := `UPDATE jamaah SET catatan=? WHERE id=?`
	var args []interface{}
	args = append(args, catatan, id)
	if brandID != nil {
		q = `UPDATE jamaah SET catatan=? WHERE id=? AND brand_id=?`
		args = append(args, *brandID)
	}
	_, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update catatan: %w", err)
	}
	return nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete menghapus jamaah. Verify brand match + tangkap FK 1451.
func (r *Repository) Delete(ctx context.Context, id int64, brandID *int64) error {
	if err := r.exists(ctx, id, brandID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM jamaah WHERE id=?`, id)
	return err
}

// ─── Relasi Kekerabatan Methods ───────────────────────────────────────────────

// ListRelasi mengambil semua relasi jamaah dua arah dan memetakan label kebalikannya.
func (r *Repository) ListRelasi(ctx context.Context, jamaahID int64, brandID *int64) ([]JamaahRelasiItem, error) {
	if err := r.exists(ctx, jamaahID, brandID); err != nil {
		return nil, err
	}

	q := `
	SELECT r.id, r.jamaah_id, r.relasi_jamaah_id, j.nama_lengkap, COALESCE(j.id_jamaah, ''), j.nik, r.hubungan, r.keterangan, r.created_at, TRUE AS is_asal
	FROM jamaah_relasi r
	JOIN jamaah j ON j.id = r.relasi_jamaah_id
	WHERE r.jamaah_id = ?
	UNION ALL
	SELECT r.id, r.relasi_jamaah_id, r.jamaah_id, j.nama_lengkap, COALESCE(j.id_jamaah, ''), j.nik, r.hubungan, r.keterangan, r.created_at, FALSE AS is_asal
	FROM jamaah_relasi r
	JOIN jamaah j ON j.id = r.jamaah_id
	WHERE r.relasi_jamaah_id = ?
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, q, jamaahID, jamaahID)
	if err != nil {
		return nil, fmt.Errorf("list relasi: %w", err)
	}
	defer rows.Close()

	items := make([]JamaahRelasiItem, 0)
	for rows.Next() {
		var item JamaahRelasiItem
		var rawHubungan string
		var nikNull sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.JamaahID,
			&item.RelasiJamaahID,
			&item.NamaRelasi,
			&item.IDJamaahRelasi,
			&nikNull,
			&rawHubungan,
			&item.Keterangan,
			&item.CreatedAt,
			&item.IsAsal,
		); err != nil {
			return nil, fmt.Errorf("scan relasi: %w", err)
		}

		if nikNull.Valid {
			item.NIKRelasi = &nikNull.String
		}
		item.HubunganAsli = rawHubungan
		if item.IsAsal {
			item.Hubungan = rawHubungan
		} else {
			if inv, ok := HubunganInverseMap[rawHubungan]; ok {
				item.Hubungan = inv
			} else {
				item.Hubungan = rawHubungan
			}
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// CreateRelasi menambahkan relasi baru dengan validasi brand, pencegahan diri sendiri, dan pencegahan duplikasi dua arah.
func (r *Repository) CreateRelasi(ctx context.Context, jamaahID int64, brandID *int64, req *CreateRelasiRequest) (*JamaahRelasiItem, error) {
	if jamaahID == req.RelasiJamaahID {
		return nil, errors.New("tidak dapat menambahkan relasi ke diri sendiri")
	}

	isValidHubungan := false
	for _, h := range ValidHubunganList {
		if h == req.Hubungan {
			isValidHubungan = true
			break
		}
	}
	if !isValidHubungan {
		return nil, errors.New("hubungan tidak valid")
	}

	// 1. Validasi jamaah subjek
	var subjekBrandID int64
	var subjekNama string
	err := r.db.QueryRowContext(ctx, "SELECT brand_id, nama_lengkap FROM jamaah WHERE id = ?", jamaahID).Scan(&subjekBrandID, &subjekNama)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check subjek jamaah: %w", err)
	}

	// Verify brand scoping jika scoped admin
	if brandID != nil && *brandID != subjekBrandID {
		return nil, ErrNotFound
	}

	// 2. Validasi jamaah target
	var targetBrandID int64
	var targetNama string
	var targetIDJamaah sql.NullString
	var targetNIK sql.NullString
	err = r.db.QueryRowContext(ctx, "SELECT brand_id, nama_lengkap, id_jamaah, nik FROM jamaah WHERE id = ?", req.RelasiJamaahID).Scan(
		&targetBrandID, &targetNama, &targetIDJamaah, &targetNIK,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("jamaah target tidak ditemukan")
	}
	if err != nil {
		return nil, fmt.Errorf("check target jamaah: %w", err)
	}

	// Pastikan 1 brand yang sama
	if subjekBrandID != targetBrandID {
		return nil, errors.New("hanya dapat menambahkan relasi antar jamaah dalam brand yang sama")
	}

	// 3. Validasi duplikasi dua arah (A-B atau B-A)
	var (
		existingID       int64
		existingA        int64
		existingB        int64
		existingHubungan string
	)
	checkQ := `SELECT id, jamaah_id, relasi_jamaah_id, hubungan FROM jamaah_relasi
		WHERE (jamaah_id = ? AND relasi_jamaah_id = ?)
		   OR (jamaah_id = ? AND relasi_jamaah_id = ?)
		LIMIT 1`
	err = r.db.QueryRowContext(ctx, checkQ, jamaahID, req.RelasiJamaahID, req.RelasiJamaahID, jamaahID).Scan(
		&existingID, &existingA, &existingB, &existingHubungan,
	)
	if err == nil {
		// Sudah ada relasi
		displayHubungan := existingHubungan
		if existingB == jamaahID {
			if inv, ok := HubunganInverseMap[existingHubungan]; ok {
				displayHubungan = inv
			}
		}
		return nil, fmt.Errorf("Relasi antara %s dan %s sudah tercatat sebagai %s", subjekNama, targetNama, displayHubungan)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing relasi: %w", err)
	}

	// 4. Insert relasi
	insertQ := `INSERT INTO jamaah_relasi (jamaah_id, relasi_jamaah_id, hubungan, keterangan) VALUES (?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, insertQ, jamaahID, req.RelasiJamaahID, req.Hubungan, req.Keterangan)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, errors.New("relasi sudah terdaftar")
		}
		return nil, fmt.Errorf("insert relasi: %w", err)
	}

	insertID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	item := &JamaahRelasiItem{
		ID:             insertID,
		JamaahID:       jamaahID,
		RelasiJamaahID: req.RelasiJamaahID,
		NamaRelasi:     targetNama,
		IDJamaahRelasi: targetIDJamaah.String,
		Hubungan:       req.Hubungan,
		HubunganAsli:   req.Hubungan,
		IsAsal:         true,
		Keterangan:     req.Keterangan,
		CreatedAt:      time.Now(),
	}
	if targetNIK.Valid {
		item.NIKRelasi = &targetNIK.String
	}

	return item, nil
}

// UpdateRelasi memperbarui kolom hubungan dan keterangan pada baris relasi yang sudah ada.
func (r *Repository) UpdateRelasi(ctx context.Context, jamaahID int64, relasiID int64, brandID *int64, req *UpdateRelasiRequest) (*JamaahRelasiItem, error) {
	if err := r.exists(ctx, jamaahID, brandID); err != nil {
		return nil, err
	}

	isValidHubungan := false
	for _, h := range ValidHubunganList {
		if h == req.Hubungan {
			isValidHubungan = true
			break
		}
	}
	if !isValidHubungan {
		return nil, errors.New("hubungan tidak valid")
	}

	var (
		rowJamaahID       int64
		rowRelasiJamaahID int64
		rowCreatedAt      time.Time
	)
	qSelect := `SELECT jamaah_id, relasi_jamaah_id, created_at FROM jamaah_relasi WHERE id = ? AND (jamaah_id = ? OR relasi_jamaah_id = ?)`
	err := r.db.QueryRowContext(ctx, qSelect, relasiID, jamaahID, jamaahID).Scan(&rowJamaahID, &rowRelasiJamaahID, &rowCreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check relasi: %w", err)
	}

	isAsal := (rowJamaahID == jamaahID)
	rawDBHubungan := req.Hubungan
	if !isAsal {
		if inv, ok := HubunganInverseMap[req.Hubungan]; ok {
			rawDBHubungan = inv
		}
	}

	qUpdate := `UPDATE jamaah_relasi SET hubungan = ?, keterangan = ? WHERE id = ?`
	_, err = r.db.ExecContext(ctx, qUpdate, rawDBHubungan, req.Keterangan, relasiID)
	if err != nil {
		return nil, fmt.Errorf("update relasi: %w", err)
	}

	var opponentID int64
	if isAsal {
		opponentID = rowRelasiJamaahID
	} else {
		opponentID = rowJamaahID
	}

	var (
		targetNama     string
		targetIDJamaah sql.NullString
		targetNIK      sql.NullString
	)
	err = r.db.QueryRowContext(ctx, "SELECT nama_lengkap, id_jamaah, nik FROM jamaah WHERE id = ?", opponentID).Scan(&targetNama, &targetIDJamaah, &targetNIK)
	if err != nil {
		return nil, fmt.Errorf("fetch opponent jamaah: %w", err)
	}

	item := &JamaahRelasiItem{
		ID:             relasiID,
		JamaahID:       jamaahID,
		RelasiJamaahID: opponentID,
		NamaRelasi:     targetNama,
		IDJamaahRelasi: targetIDJamaah.String,
		Hubungan:       req.Hubungan,
		HubunganAsli:   rawDBHubungan,
		IsAsal:         isAsal,
		Keterangan:     req.Keterangan,
		CreatedAt:      rowCreatedAt,
	}
	if targetNIK.Valid {
		item.NIKRelasi = &targetNIK.String
	}

	return item, nil
}

// DeleteRelasi menghapus baris relasi baik subjek sebagai pembuat maupun penerima.
func (r *Repository) DeleteRelasi(ctx context.Context, jamaahID int64, relasiID int64, brandID *int64) error {
	if err := r.exists(ctx, jamaahID, brandID); err != nil {
		return err
	}

	q := `DELETE FROM jamaah_relasi WHERE id = ? AND (jamaah_id = ? OR relasi_jamaah_id = ?)`
	res, err := r.db.ExecContext(ctx, q, relasiID, jamaahID, jamaahID)
	if err != nil {
		return fmt.Errorf("delete relasi: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) exists(ctx context.Context, id int64, brandID *int64) error {
	var count int
	q := `SELECT COUNT(*) FROM jamaah WHERE id=?`
	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	if err != nil {
		return fmt.Errorf("jamaah.exists: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
