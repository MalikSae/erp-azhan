package payment

import (
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Keputusan 11 November 2026: transfer yang dicatat admin wajib berbukti; tunai tidak.
func TestPesanBuktiWajib(t *testing.T) {
	bukti, kosong := "/uploads/payment-proofs/a.jpg", ""
	cases := []struct {
		metode string
		bukti  *string
		tolak  bool
	}{
		{metodeTransfer, nil, true},
		{metodeTransfer, &kosong, true},
		{metodeTransfer, &bukti, false},
		{metodeTunai, nil, false},
		{metodeTunai, &bukti, false},
	}
	for _, c := range cases {
		if got := pesanBuktiWajib(c.metode, c.bukti) != ""; got != c.tolak {
			t.Errorf("metode %s bukti %v: tolak=%v, mau %v", c.metode, c.bukti, got, c.tolak)
		}
	}
}

// Kolom dicatat_oleh terbaca lewat kolomPayment; pembayaran tanpa admin pencatat (portal) bernilai NULL.
func TestDicatatOlehTerbaca(t *testing.T) {
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, Blocked: true})
	testdb.Exec(t, tx, `INSERT INTO admin_users (email, password_hash) VALUES ('pencatat-uji@uji.test', 'x')`)
	var admin int64
	if err := tx.QueryRow(`SELECT id FROM admin_users WHERE email='pencatat-uji@uji.test'`).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `INSERT INTO payments (booking_id, jumlah, status, source, dicatat_oleh, verified_by) VALUES (?,?,?,?,?,?)`,
		b.ID, 500, StatusConfirmed, SourceAdmin, admin, admin)
	portal := testdb.AddPayment(t, tx, b.ID, 100, StatusPending)

	var p Payment
	if err := tx.QueryRow(`SELECT `+kolomPayment+` FROM payments p WHERE p.booking_id=? AND p.dicatat_oleh IS NOT NULL`, b.ID).Scan(tujuanScan(&p)...); err != nil {
		t.Fatal(err)
	}
	if p.DicatatOleh == nil || *p.DicatatOleh != admin || p.VerifiedBy == nil || *p.VerifiedBy != admin {
		t.Fatalf("pencatat = %v, verifikator = %v, mau %d", p.DicatatOleh, p.VerifiedBy, admin)
	}
	var pencatatPortal sql.NullInt64
	if err := tx.QueryRow(`SELECT dicatat_oleh FROM payments WHERE id=?`, portal).Scan(&pencatatPortal); err != nil {
		t.Fatal(err)
	}
	if pencatatPortal.Valid {
		t.Fatalf("pembayaran tanpa admin pencatat harus NULL, dapat %d", pencatatPortal.Int64)
	}
}
