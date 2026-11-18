package customer

import (
	"context"
	"errors"
)

// Akses akun atas data jamaah (keputusan 11 November 2026).
//
// Jamaah dipakai ulang lintas akun lewat NIK + nama, jadi memiliki booking yang memuat seorang jamaah
// belum berarti berhak atas data lengkapnya. Akses penuh hanya untuk akun pembuat jamaah
// (jamaah.dibuat_oleh_akun_id) dan akun yang tertaut ke jamaah itu (aktivasi). Akun lain mendapat akses terbatas: nama, NIK tersamar, jenis pax, dan
// angka kelengkapan; tidak boleh mengubah data; hanya boleh mengunggah jenis dokumen yang belum ada.

// sqlAksesPenuh: ekspresi SQL bernilai benar bila akun adalah pembuat jamaah, atau akun itu tertaut ke
// jamaah tersebut (customer_accounts.jamaah_id, lahir dari aktivasi; keputusan 18 November 2026: jamaah
// berhak penuh atas data dirinya). Memakai alias tabel j dan satu parameter (id akun). Satu-satunya tempat
// aturan akses penuh dituliskan.
const sqlAksesPenuh = `EXISTS (SELECT 1 FROM customer_accounts ca_ak WHERE ca_ak.id = ? AND (j.dibuat_oleh_akun_id = ca_ak.id OR ca_ak.jamaah_id = j.id))`

const pesanAksesTerbatas = "Data jamaah ini sudah terdaftar. Hubungi admin untuk perubahan."

var errAksesTerbatas = errors.New(pesanAksesTerbatas)

// aksesPenuhJamaah melaporkan apakah akun berhak atas data lengkap satu jamaah.
func aksesPenuhJamaah(ctx context.Context, q dbq, akunID, jamaahID int64) (bool, error) {
	var penuh bool
	err := q.QueryRowContext(ctx, `SELECT `+sqlAksesPenuh+` FROM jamaah j WHERE j.id = ?`, akunID, jamaahID).Scan(&penuh)
	return penuh, err
}

// pesanBukanDiriSendiri: akun peran jamaah mencoba bertindak atas jamaah lain di booking yang sama.
const pesanBukanDiriSendiri = "Hanya untuk data Anda sendiri."
