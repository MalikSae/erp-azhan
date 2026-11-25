package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"erp-azhan/api/internal/booking"
)

// ubahTanggalBerangkatTx menegakkan aturan ubah tanggal berangkat (audit menu Paket S-04). Paket tanpa
// booking aktif bebas diubah. Bila ada booking aktif: paket yang sudah berangkat terkunci, tanggal baru
// tidak boleh lampau, alasan wajib diisi, dan perubahan dicatat di schedule_tanggal_log. Dipanggil setelah
// baris jadwal dikunci, sehingga booking yang masuk bersamaan tidak lolos dari hitungan.
func ubahTanggalBerangkatTx(ctx context.Context, tx *sql.Tx, id int64, lama string, inp *ScheduleInput) error {
	if inp.BerangkatTanggal == lama {
		return nil
	}
	aktif, err := booking.HitungAktif(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("schedule: hitung booking aktif: %w", err)
	}
	if aktif == 0 {
		return nil
	}
	hariIni := todayWIB()
	if lama < hariIni {
		return fmt.Errorf("%w: paket sudah berangkat dan punya %d booking aktif, tanggal berangkat tidak dapat diubah", ErrConflict, aktif)
	}
	if inp.BerangkatTanggal < hariIni {
		return fmt.Errorf("%w: paket punya %d booking aktif, tanggal berangkat baru tidak boleh sudah lewat", ErrConflict, aktif)
	}
	if inp.AlasanUbahTanggal == "" {
		return fmt.Errorf("%w: paket punya %d booking aktif, alasan ubah tanggal berangkat wajib diisi", ErrConflict, aktif)
	}
	var oleh any
	if inp.ActorID > 0 {
		oleh = inp.ActorID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schedule_tanggal_log (schedule_id, tanggal_lama, tanggal_baru, booking_aktif, alasan, diubah_oleh)
		VALUES (?, ?, ?, ?, ?, ?)`, id, lama, inp.BerangkatTanggal, aktif, inp.AlasanUbahTanggal, oleh); err != nil {
		return fmt.Errorf("schedule: catat ubah tanggal: %w", err)
	}
	return nil
}

// todayWIB mengembalikan tanggal hari ini (YYYY-MM-DD) zona bisnis.
func todayWIB() string {
	return time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02")
}

// batasPublikWIB: tanggal berangkat paling awal yang masih tampil dan bisa dipesan di situs publik (hari ini + 30 hari).
func batasPublikWIB() string {
	return time.Now().In(time.FixedZone("WIB", 7*60*60)).AddDate(0, 0, 30).Format("2006-01-02")
}
