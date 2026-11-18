package customer

import (
	"context"
	"database/sql"
	"log"
	"net/http"
)

// Pandangan jamaah (ADR 0005): akun yang tertaut ke jamaah (customer_accounts.jamaah_id, lahir dari aktivasi)
// melihat perjalanannya yang dibuat orang lain, hanya baca dan terbatas. owned() tetap pemilik-saja; semua
// handler tulis dan pembayaran tidak ikut berubah.

// perjalananJamaah: satu kartu per booking orang lain yang memuat jamaah tertaut. Daftar putih: tidak ada
// kode booking, pax lain, harga, pembayaran, rekening, atau kantor.
type perjalananJamaah struct {
	BookingID int64           `json:"booking_id"`
	Paket     string          `json:"paket"`
	Berangkat string          `json:"berangkat"`
	Status    string          `json:"status"` // baru | dp | lunas
	Pax       []paxPerjalanan `json:"pax"`
	CanPay    bool            `json:"can_pay"`
	CanCancel bool            `json:"can_cancel"`
}

type paxPerjalanan struct {
	JamaahID int64  `json:"jamaah_id"`
	Nama     string `json:"nama"`
	PaxType  string `json:"pax_type"`
}

// sqlBookingPeran: booking akun lain yang aktif (bukan draft, bukan batal, bukan baru yang tahannya lewat)
// dan memuat jamaah tertaut. Parameter: id akun, id jamaah tertaut.
const sqlBookingPeran = `
	FROM booking_pax bp
	JOIN bookings b ON b.id = bp.booking_id
	JOIN schedules s ON s.id = b.schedule_id
	JOIN jamaah j ON j.id = bp.jamaah_id
	WHERE bp.jamaah_id = ? AND bp.pax_status = 'aktif'
	  AND NOT (b.customer_account_id <=> ?)
	  AND b.status NOT IN ('draft','batal')
	  AND (b.status <> 'baru' OR (b.is_seat_blocked AND (b.seat_hold_expires_at IS NULL OR b.seat_hold_expires_at > NOW())))`

func statusPeran(s string) string {
	switch s {
	case "baru", "dp":
		return s
	default:
		return "lunas" // lunas, dokumen_lengkap, siap_berangkat
	}
}

func jamaahTertaut(ctx context.Context, q dbq, akunID int64) (int64, bool, error) {
	var id sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT jamaah_id FROM customer_accounts WHERE id = ?`, akunID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id.Int64, id.Valid, nil
}

// muatPerjalananJamaah memuat kartu perjalanan peran jamaah; kosong bila akun tidak tertaut.
func muatPerjalananJamaah(ctx context.Context, q dbq, akunID int64) ([]perjalananJamaah, error) {
	jid, ok, err := jamaahTertaut(ctx, q, akunID)
	if err != nil || !ok {
		return []perjalananJamaah{}, err
	}
	rows, err := q.QueryContext(ctx, `
		SELECT b.id, s.jadwal_nama, COALESCE(DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'),''), b.status, j.id, j.nama_lengkap, bp.pax_type`+
		sqlBookingPeran+` ORDER BY b.id DESC, bp.id`, jid, akunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []perjalananJamaah{}
	for rows.Next() {
		var p perjalananJamaah
		var status string
		var x paxPerjalanan
		if err := rows.Scan(&p.BookingID, &p.Paket, &p.Berangkat, &status, &x.JamaahID, &x.Nama, &x.PaxType); err != nil {
			return nil, err
		}
		p.Status = statusPeran(status)
		p.Pax = []paxPerjalanan{x}
		out = append(out, p)
	}
	return out, rows.Err()
}

// muatAnggotaPeran: baris Data Jamaah dari booking orang lain, hanya baris pax milik jamaah tertaut.
func muatAnggotaPeran(ctx context.Context, q dbq, akunID int64) ([]anggotaJamaah, error) {
	jid, ok, err := jamaahTertaut(ctx, q, akunID)
	if err != nil || !ok {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, COALESCE(DATE_FORMAT(j.tanggal_lahir,'%Y-%m-%d'),''), bp.pax_type,
		       b.id, b.id_booking, b.status, s.jadwal_nama, COALESCE(DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'),''),
		       `+sqlAksesPenuh+sqlBookingPeran+` ORDER BY b.id DESC`, akunID, jid, akunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []anggotaJamaah
	for rows.Next() {
		var a anggotaJamaah
		if err := rows.Scan(&a.ID, &a.Nama, &a.Lahir, &a.PaxType, &a.BookingID, &a.BookingKode, &a.StatusBooking, &a.Paket, &a.Berangkat, &a.Penuh); err != nil {
			return nil, err
		}
		a.DariPeran = true
		out = append(out, a)
	}
	return out, rows.Err()
}

// PerjalananJamaah GET /api/account/perjalanan-jamaah: hanya baca. Akun tidak tertaut menerima daftar kosong.
func (o *Orders) PerjalananJamaah(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	items, err := muatPerjalananJamaah(r.Context(), o.db, acc.ID)
	if err != nil {
		log.Printf("[ERROR] account perjalanan jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil perjalanan.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"perjalanan": items})
}
