package customer

import (
	"context"
	"errors"
)

// Akses akun atas data jamaah (keputusan 11 November 2026).
//
// Jamaah dipakai ulang lintas akun lewat NIK + nama, jadi memiliki booking yang memuat seorang jamaah
// belum berarti berhak atas data lengkapnya. Akses penuh hanya untuk akun pembuat jamaah
// (jamaah.dibuat_oleh_akun_id). Akun lain mendapat akses terbatas: nama, NIK tersamar, jenis pax, dan
// angka kelengkapan; tidak boleh mengubah data; hanya boleh mengunggah jenis dokumen yang belum ada.

// sqlAksesPenuh: ekspresi SQL bernilai benar bila akun adalah pembuat jamaah. Memakai alias tabel j dan
// satu parameter (id akun). Satu-satunya tempat aturan akses penuh dituliskan.
const sqlAksesPenuh = `(j.dibuat_oleh_akun_id <=> ?)`

const pesanAksesTerbatas = "Data jamaah ini sudah terdaftar. Hubungi admin untuk perubahan."

var errAksesTerbatas = errors.New(pesanAksesTerbatas)

// aksesPenuhJamaah melaporkan apakah akun berhak atas data lengkap satu jamaah.
func aksesPenuhJamaah(ctx context.Context, q dbq, akunID, jamaahID int64) (bool, error) {
	var penuh bool
	err := q.QueryRowContext(ctx, `SELECT `+sqlAksesPenuh+` FROM jamaah j WHERE j.id = ?`, akunID, jamaahID).Scan(&penuh)
	return penuh, err
}
