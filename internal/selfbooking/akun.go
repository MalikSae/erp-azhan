package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

// resolvePIC menentukan jamaah PIC (pemesan yang ikut berangkat): dicocokkan lewat NIK dan nama seperti
// penumpang lain (resolveAnggotaPublik), bukan lagi dibuat baru tanpa NIK (spek 10 butir 5). Akun tidak
// terhubung ke satu jamaah; satu orang dikenali dari NIK-nya, siapa pun akun yang memasukkannya.
//
// Nomor WhatsApp dan email PIC diisi handler dari tabel akun, bukan dari payload, sehingga PIC tidak bisa
// diarahkan ke kontak orang lain. Jamaah lama yang cocok tidak ditimpa datanya; email akun hanya melengkapi
// jamaah yang emailnya masih kosong.
func resolvePIC(ctx context.Context, tx *sql.Tx, brandID, accountID int64, pic PICInput) (int64, error) {
	var accBrand int64
	err := tx.QueryRowContext(ctx, `SELECT brand_id FROM customer_accounts WHERE id=? FOR UPDATE`, accountID).Scan(&accBrand)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && accBrand != brandID) {
		return 0, ErrAkunTidakValid
	}
	if err != nil {
		return 0, fmt.Errorf("baca akun: %w", err)
	}
	var hp *string
	if pic.NoHP != "" {
		hp = &pic.NoHP
	}
	id, err := resolveAnggotaPublik(ctx, tx, brandID, accountID, AnggotaInput{
		PaxType: "reguler", NamaLengkap: strings.TrimSpace(pic.NamaLengkap), NoHP: hp, JenisKelamin: pic.JenisKelamin, NIK: pic.NIK, Nomor: 1,
	})
	if err != nil {
		return 0, err
	}
	if pic.Email != nil && strings.TrimSpace(*pic.Email) != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET email=? WHERE id=? AND (email IS NULL OR email='')`, strings.TrimSpace(*pic.Email), id); err != nil {
			return 0, fmt.Errorf("lengkapi email PIC: %w", err)
		}
	}
	return id, nil
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
