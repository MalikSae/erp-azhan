package selfbooking

import (
	"context"
	"database/sql"
	"testing"
)

// Akun pemesan dicatat sebagai pembuat hanya untuk jamaah yang baru dibuat. Jamaah lama yang dipakai
// ulang lewat NIK + nama tidak berpindah pembuat (code review endpoint publik, 11 November 2026, C1).
func TestResolveAnggotaPublikMencatatPembuat(t *testing.T) {
	tx, _, brandID, agen := setupAgenBooking(t)
	ctx := context.Background()
	pembuat := func(id int64) sql.NullInt64 {
		var v sql.NullInt64
		if err := tx.QueryRow(`SELECT dibuat_oleh_akun_id FROM jamaah WHERE id=?`, id).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}

	nik := "3174010101018888"
	baru, err := resolveAnggotaPublik(ctx, tx, brandID, agen, AnggotaInput{PaxType: "reguler", NamaLengkap: "Jamaah Pembuat Uji", JenisKelamin: "L", NIK: &nik})
	if err != nil {
		t.Fatal(err)
	}
	if got := pembuat(baru); got.Int64 != agen {
		t.Fatalf("jamaah baru: pembuat = %v, mau %d", got, agen)
	}

	nikAdmin := "3174010101017777"
	dariAdmin, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Jamaah Admin Uji", JenisKelamin: "P", NIK: &nikAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if got := pembuat(dariAdmin); got.Valid {
		t.Fatalf("tanpa akun pemesan pembuat harus kosong, dapat %d", got.Int64)
	}

	ulang, err := resolveAnggotaPublik(ctx, tx, brandID, agen, AnggotaInput{PaxType: "reguler", NamaLengkap: "Jamaah Admin Uji", JenisKelamin: "P", NIK: &nikAdmin})
	if err != nil || ulang != dariAdmin {
		t.Fatalf("pakai ulang: id %d, %v", ulang, err)
	}
	if got := pembuat(dariAdmin); got.Valid {
		t.Fatalf("pakai ulang tidak boleh mengklaim pembuat, dapat %d", got.Int64)
	}
}
