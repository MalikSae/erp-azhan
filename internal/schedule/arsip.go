package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ArsipkanLampau mengubah paket berstatus published yang tanggal berangkatnya sudah lewat (WIB)
// menjadi archived. Aturan yang sama dengan pengarsipan manual: sesudah tanggal berangkat lewat,
// paket boleh diarsipkan walau masih punya booking. Mengembalikan jumlah paket yang diarsipkan.
func (r *Repository) ArsipkanLampau(ctx context.Context) (int64, error) {
	return arsipkanLampau(ctx, r.db)
}

// execer dipenuhi *sql.DB dan *sql.Tx, supaya logika yang sama bisa diuji di dalam transaksi.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func arsipkanLampau(ctx context.Context, ex execer) (int64, error) {
	res, err := ex.ExecContext(ctx, `UPDATE schedules SET status='archived' WHERE status='published' AND berangkat_tanggal < ?`, todayWIB())
	if err != nil {
		return 0, fmt.Errorf("schedule.ArsipkanLampau: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// JalankanArsipOtomatis mengarsipkan paket lampau sekali saat dipanggil, lalu sekali tiap `jeda`
// sampai ctx dibatalkan. Dipanggil sebagai goroutine saat API menyala.
func JalankanArsipOtomatis(ctx context.Context, r *Repository, jeda time.Duration) {
	jalan := func() {
		n, err := r.ArsipkanLampau(ctx)
		if err != nil {
			log.Printf("[ERROR] arsip otomatis paket: %v", err)
			return
		}
		if n > 0 {
			log.Printf("[INFO] arsip otomatis paket: %d paket lampau diarsipkan", n)
		}
	}
	jalan()
	t := time.NewTicker(jeda)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			jalan()
		}
	}
}
