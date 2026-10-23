package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"erp-azhan/api/internal/shared"
)

// Batas akun pelanggan (K2). Setiap booking menahan seat 24 jam tanpa pembayaran, jadi jumlah seat
// yang ditahan booking "baru" milik satu akun dibatasi. Dihitung dari database di dalam transaksi
// booking dengan baris akun dikunci (FOR UPDATE), sehingga permintaan paralel tidak bisa melewatinya.
const (
	maxSeatAktifPerJadwalAkun = 9  // seat ditahan (booking baru, belum lewat batas tahan) per jadwal per akun
	maxSeatAktifPerAkun       = 18 // total seat ditahan per akun di semua jadwal
)

var (
	ErrBatasSeatAkun      = errors.New("batas seat yang ditahan akun Anda sudah tercapai. Selesaikan pembayaran booking yang masih menunggu atau tunggu masa tahan seat berakhir, lalu coba lagi")
	ErrBatasBookingHarian = errors.New("akun ini sudah mencapai batas booking hari ini, silakan hubungi admin travel")
)

type batasAkun struct {
	seatPerJadwal int
	seatTotal     int
	bookingHarian int // booking dibuat per 24 jam
}

var batasAkunDefault = batasAkun{seatPerJadwal: maxSeatAktifPerJadwalAkun, seatTotal: maxSeatAktifPerAkun, bookingHarian: bookingAccountSuccessLimit}

// cekBatasAkun mengunci baris akun lalu memeriksa batas booking harian dan seat yang masih ditahan.
func cekBatasAkun(ctx context.Context, tx *sql.Tx, accountID, scheduleID int64, butuhSeat int, b batasAkun) error {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM customer_accounts WHERE id=? FOR UPDATE`, accountID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAkunTidakValid
	}
	if err != nil {
		return fmt.Errorf("kunci akun: %w", err)
	}
	now := time.Now()
	var dibuat int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings WHERE customer_account_id=? AND status<>'draft' AND created_at > ?`,
		accountID, now.Add(-24*time.Hour)).Scan(&dibuat); err != nil {
		return fmt.Errorf("hitung booking akun: %w", err)
	}
	if dibuat >= b.bookingHarian {
		return ErrBatasBookingHarian
	}
	var total, diJadwal int
	if err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(seat_count),0), COALESCE(SUM(CASE WHEN schedule_id=? THEN seat_count ELSE 0 END),0)
		FROM bookings
		WHERE customer_account_id=? AND status='baru' AND is_seat_blocked=TRUE
		  AND (seat_hold_expires_at IS NULL OR seat_hold_expires_at > ?)`,
		scheduleID, accountID, now).Scan(&total, &diJadwal); err != nil {
		return fmt.Errorf("hitung seat akun: %w", err)
	}
	if diJadwal+butuhSeat > b.seatPerJadwal || total+butuhSeat > b.seatTotal {
		return ErrBatasSeatAkun
	}
	return nil
}

func nullableID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

// resolveAkunPIC menentukan jamaah PIC untuk booking atas nama akun pelanggan.
//
//  1. Akun sudah punya jamaah_id: pakai itu.
//  2. Belum ada: SELALU buat jamaah baru dari data akun, walau nomor WhatsApp akun sama
//     dengan jamaah lama di brand ini.
//
// Keputusan K1 (tinjauan keamanan 2026-10-22): nomor HP akun tidak diverifikasi, jadi
// kecocokan HP (dan nama) bukan bukti identitas. Menautkan akun ke jamaah lama membuka data
// jamaah itu (NIK, jenis kelamin) kepada siapa pun yang tahu nomornya, dan jawaban berbeda
// untuk nama cocok/beda menjadi oracle. Akibatnya pelanggan lama bisa punya dua baris jamaah;
// admin menggabungkannya secara manual (pengamanan > deduplikasi otomatis).
//
// Nomor, email, dan identitas akun diambil dari tabel akun, bukan dari request,
// sehingga PIC tidak bisa diarahkan ke nomor orang lain lewat payload.
func resolveAkunPIC(ctx context.Context, tx *sql.Tx, brandID, accountID int64, pic PICInput) (int64, error) {
	var accBrand int64
	var email, noHP string
	var jamaahID sql.NullInt64
	err := tx.QueryRowContext(ctx,
		`SELECT brand_id, email, no_hp, jamaah_id FROM customer_accounts WHERE id=? FOR UPDATE`, accountID,
	).Scan(&accBrand, &email, &noHP, &jamaahID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && accBrand != brandID) {
		return 0, ErrAkunTidakValid
	}
	if err != nil {
		return 0, fmt.Errorf("baca akun: %w", err)
	}
	if jamaahID.Valid {
		return jamaahID.Int64, nil
	}

	// Skema menegakkan satu nomor HP = satu jamaah per brand (uq_jamaah_brand_hp, migrasi 060).
	// Bila nomor akun sudah dipakai jamaah lain, jamaah PIC baru dibuat TANPA nomor HP (kontak
	// pemesan tetap tercatat di akun); jawabannya sama persis dengan nomor yang belum dipakai.
	hpPIC := noHP
	canonical, local := shared.PhoneVariants(noHP)
	var dipakai int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM jamaah
		WHERE brand_id = ? AND REGEXP_REPLACE(COALESCE(no_hp,''),'[^0-9]','') IN (?,?)
		ORDER BY id LIMIT 1 FOR UPDATE`, brandID, canonical, local).Scan(&dipakai)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return 0, fmt.Errorf("periksa nomor pemesan: %w", err)
	default:
		hpPIC = ""
		log.Printf("[AUDIT] selfbooking hp_pic_bentrok brand=%d akun=%d jamaah_lama=%d (PIC baru tanpa nomor; tinjau/gabungkan manual)", brandID, accountID, dipakai)
	}

	jk := pic.JenisKelamin
	em := email
	// CreateJamaah, bukan ResolveJamaah: ResolveJamaah memakai ulang baris lama yang cocok nomor HP.
	baruID, err := shared.CreateJamaah(ctx, tx, brandID, shared.JamaahInput{
		NamaLengkap:  strings.TrimSpace(pic.NamaLengkap),
		NoHP:         hpPIC,
		Email:        &em,
		JenisKelamin: &jk,
	})
	if err != nil {
		return 0, fmt.Errorf("buat jamaah pemesan: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, baruID, accountID); err != nil {
		return 0, fmt.Errorf("kaitkan akun ke jamaah: %w", err)
	}
	return baruID, nil
}

// simpanTravelers menyimpan anggota rombongan ke daftar jamaah tersimpan akun
// (tanpa duplikat menurut nama + tanggal lahir).
func simpanTravelers(ctx context.Context, tx *sql.Tx, accountID int64, anggota []AnggotaInput) error {
	for _, a := range anggota {
		nama := strings.TrimSpace(a.NamaLengkap)
		if nama == "" {
			continue
		}
		var dob any
		if a.TanggalLahir != nil && strings.TrimSpace(*a.TanggalLahir) != "" {
			dob = strings.TrimSpace(*a.TanggalLahir)
		}
		var nik any
		if n := nikBersih(a.NIK); n != "" {
			nik = n
		}
		var hp any
		if a.NoHP != nil && strings.TrimSpace(*a.NoHP) != "" {
			hp = strings.TrimSpace(*a.NoHP)
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO customer_travelers (account_id, nama_lengkap, nama_normal, jenis_kelamin, no_hp, tanggal_lahir, nik)
			SELECT ?, ?, ?, ?, ?, ?, ? FROM DUAL
			WHERE NOT EXISTS (
				SELECT 1 FROM customer_travelers WHERE account_id=? AND nama_normal=? AND tanggal_lahir <=> ?
			)`,
			accountID, nama, normalizeName(nama), a.JenisKelamin, hp, dob, nik,
			accountID, normalizeName(nama), dob)
		if err != nil {
			return fmt.Errorf("simpan jamaah tersimpan: %w", err)
		}
		if nik != nil { // jamaah tersimpan lama yang belum punya NIK dilengkapi
			if _, err = tx.ExecContext(ctx, `UPDATE customer_travelers SET nik=? WHERE account_id=? AND nama_normal=? AND tanggal_lahir <=> ? AND nik IS NULL`, nik, accountID, normalizeName(nama), dob); err != nil {
				return fmt.Errorf("lengkapi nik jamaah tersimpan: %w", err)
			}
		}
	}
	return nil
}
