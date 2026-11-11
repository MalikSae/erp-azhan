package shared

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNotFound         = errors.New("data tidak ditemukan")
	ErrAmbiguousJamaah  = errors.New("lebih dari satu jamaah memiliki nomor telepon tersebut")
	ErrBrandCodeMissing = errors.New("kode_brand belum diatur")
)

const codeCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

var nonDigit = regexp.MustCompile(`[^0-9]+`)

// JamaahInput berisi data jamaah untuk proses resolusi.
type JamaahInput struct {
	ID           *int64  `json:"id,omitempty"`
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         string  `json:"no_hp"`
	Email        *string `json:"email,omitempty"`
	Alamat       *string `json:"alamat,omitempty"`
	JenisKelamin *string `json:"jenis_kelamin,omitempty"`
}

// PhoneVariants mengembalikan versi 62... dan 08... dari sebuah nomor HP.
func PhoneVariants(phone string) (string, string) {
	digits := nonDigit.ReplaceAllString(phone, "")
	if strings.HasPrefix(digits, "0") {
		return "62" + digits[1:], digits
	}
	if strings.HasPrefix(digits, "62") {
		return digits, "0" + digits[2:]
	}
	return digits, digits
}

// ResolveJamaah mencari jamaah berdasarkan ID atau No HP. Jika tidak ditemukan, akan membuat data jamaah baru.
func ResolveJamaah(ctx context.Context, tx *sql.Tx, brandID int64, input JamaahInput) (int64, error) {
	if input.ID != nil {
		var found int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM jamaah WHERE id=? AND brand_id=? FOR UPDATE`, *input.ID, brandID).Scan(&found); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrNotFound
			}
			return 0, fmt.Errorf("find jamaah: %w", err)
		}
		return found, nil
	}

	canonical, local := PhoneVariants(input.NoHP)
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM jamaah WHERE brand_id=? AND REGEXP_REPLACE(COALESCE(no_hp,''),'[^0-9]','') IN (?,?) ORDER BY id LIMIT 2 FOR UPDATE`,
		brandID, canonical, local,
	)
	if err != nil {
		return 0, fmt.Errorf("find jamaah by phone: %w", err)
	}
	var matches []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		matches = append(matches, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if len(matches) > 1 {
		return 0, ErrAmbiguousJamaah
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return CreateJamaah(ctx, tx, brandID, input)
}

// CreateJamaah selalu menyisipkan baris jamaah baru tanpa mencari baris lama yang cocok nomor HP.
// Dipakai jalur yang tidak boleh menggabungkan data hanya karena nomor HP sama (mis. PIC akun
// pelanggan: nomor akun belum terverifikasi sehingga tidak membuktikan identitas).
func CreateJamaah(ctx context.Context, tx *sql.Tx, brandID int64, input JamaahInput) (int64, error) {
	var brandCode sql.NullString
	var counter uint64
	if err := tx.QueryRowContext(ctx, `SELECT kode_brand,jamaah_counter FROM brands WHERE id=? FOR UPDATE`, brandID).Scan(&brandCode, &counter); err != nil {
		return 0, fmt.Errorf("lock brand for jamaah: %w", err)
	}
	if !brandCode.Valid || strings.TrimSpace(brandCode.String) == "" {
		return 0, ErrBrandCodeMissing
	}
	counter++
	if _, err := tx.ExecContext(ctx, `UPDATE brands SET jamaah_counter=? WHERE id=?`, counter, brandID); err != nil {
		return 0, fmt.Errorf("update jamaah counter: %w", err)
	}
	idJamaah := fmt.Sprintf("%s-%02d%02d%06d", strings.ToUpper(strings.TrimSpace(brandCode.String)), time.Now().Year()%100, int(time.Now().Month()), counter)
	kodeJamaah, err := UniqueCode(ctx, tx, "jamaah", "kode_jamaah", "", 6)
	if err != nil {
		return 0, err
	}

	// Nomor kosong disimpan NULL (bukan ''), agar tidak bentrok dengan kunci unik nomor per brand.
	var noHP any
	if strings.TrimSpace(input.NoHP) != "" {
		noHP = input.NoHP
	}
	result, err := tx.ExecContext(ctx,
		`INSERT INTO jamaah (brand_id,id_jamaah,kode_jamaah,nama_lengkap,no_hp,email,alamat,jenis_kelamin) VALUES (?,?,?,?,?,?,?,?)`,
		brandID, idJamaah, kodeJamaah, input.NamaLengkap, noHP, input.Email, input.Alamat, input.JenisKelamin,
	)
	if err != nil {
		return 0, fmt.Errorf("create jamaah: %w", err)
	}
	return result.LastInsertId()
}

// UniqueCode menghasilkan kode unik random dengan prefix.
func UniqueCode(ctx context.Context, tx *sql.Tx, table, column, prefix string, randomLength int) (string, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", table, column)
	for attempt := 0; attempt < 20; attempt++ {
		var suffix strings.Builder
		for i := 0; i < randomLength; i++ {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeCharset))))
			if err != nil {
				return "", err
			}
			suffix.WriteByte(codeCharset[n.Int64()])
		}
		code := prefix + suffix.String()
		var count int
		if err := tx.QueryRowContext(ctx, query, code).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("gagal membuat kode unik")
}

// GugurkanKlaimPembuatBooking menjalankan aturan gugur klaim pembuat untuk semua jamaah di satu booking
// (keputusan 11 November 2026). Akun pembuat jamaah (jamaah.dibuat_oleh_akun_id) kehilangan klaimnya bila
// jamaah itu dipakai booking pihak lain (admin atau akun lain) sementara akun pembuat belum pernah punya
// pembayaran terkonfirmasi pada booking yang memuat jamaah itu. Tanpa aturan ini, siapa pun bisa mendahului
// membuat jamaah dengan NIK + nama orang lain lalu membaca datanya setelah orang itu mendaftar sungguhan.
// Dipanggil setiap kali booking baru terbentuk, dari jalur mana pun.
func GugurkanKlaimPembuatBooking(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE jamaah j
		JOIN booking_pax bp ON bp.jamaah_id = j.id AND bp.booking_id = ?
		JOIN bookings b ON b.id = bp.booking_id
		SET j.dibuat_oleh_akun_id = NULL
		WHERE j.dibuat_oleh_akun_id IS NOT NULL
		  AND NOT (b.customer_account_id <=> j.dibuat_oleh_akun_id)
		  AND NOT EXISTS (
		    SELECT 1 FROM payments p
		    JOIN bookings b2 ON b2.id = p.booking_id
		    JOIN booking_pax bp2 ON bp2.booking_id = b2.id
		    WHERE bp2.jamaah_id = j.id AND b2.customer_account_id = j.dibuat_oleh_akun_id AND p.status = 'confirmed'
		  )`, bookingID)
	if err != nil {
		return fmt.Errorf("gugurkan klaim pembuat jamaah: %w", err)
	}
	return nil
}
