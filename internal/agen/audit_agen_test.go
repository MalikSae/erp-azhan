package agen

import (
	"context"
	"erp-azhan/api/internal/testdb"
	"testing"
)

// Audit probes: PASS confirms the behavior described, not correctness.
func TestAuditAgenBankAccountWithoutDigitsAccepted(t *testing.T) {
	req := AjukanPencairanRequest{Nominal: 500000, Rekening: Rekening{Bank: "BSI", Nomor: "---", AtasNama: "Audit"}}
	if err := req.normalize(); err != nil {
		t.Fatalf("probe changed: %v", err)
	}
	t.Log("CONFIRMED: bank account --- passes payout validation")
}

func TestAuditAgenNonexistentPhotoAccepted(t *testing.T) {
	_, tx := testdb.Tx(t)
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE brands SET biaya_pendaftaran_agen=500000 WHERE id=?`, brand)
	applicant := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Applicant"})
	req := AjukanRequest{FotoAgenURL: "/api/admin/media/dokumen-jamaah/00000000-0000-0000-0000-000000000000.jpg", Domisili: "Jakarta", SetujuSyaratKetentuan: true}
	if !fotoPortalURL.MatchString(req.FotoAgenURL) {
		t.Fatal("fixture must pass HTTP URL validation")
	}
	if err := ajukanTx(context.Background(), tx, applicant, req); err != nil {
		t.Fatal(err)
	}
	var photo string
	if err := tx.QueryRow(`SELECT foto_agen_url FROM jamaah WHERE id=?`, applicant).Scan(&photo); err != nil {
		t.Fatal(err)
	}
	if photo != req.FotoAgenURL {
		t.Fatalf("probe changed: photo=%s", photo)
	}
	t.Log("CONFIRMED: invented photo URL accepted without uploading a file")
}
