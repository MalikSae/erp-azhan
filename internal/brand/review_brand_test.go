package brand

import (
	"encoding/json"
	"testing"
)

// Code review Paket dan master, 11 November 2026: nominal brand harus bilangan bulat rupiah (ADR-0001).
func TestSemuaRupiahBulat(t *testing.T) {
	if !semuaRupiahBulat() || !semuaRupiahBulat(0, 500_000, 100_000) {
		t.Fatal("nominal bulat harus lolos")
	}
	if semuaRupiahBulat(500_000, 100_000.5) {
		t.Fatal("nominal pecahan harus ditolak")
	}
}

// Keputusan 11 November 2026: nominal brand yang tidak dikirim tidak diubah, bukan disimpan 0.
func TestPertahankanNominalKosong(t *testing.T) {
	lama := &Brand{MinimalDP: 5_000_000, CashbackAlumni: 250_000, BiayaPendaftaranAgen: 300_000, UjrohClosing: 500_000, UjrohPembinaan: 100_000, UjrohRepeatBooking: 50_000}

	// Hanya ujroh_closing yang dikirim (bernilai 0 dengan sengaja): yang lain mengikuti nilai tersimpan.
	badan := []byte(`{"name":"Hana","ujroh_closing":0}`)
	var req UpdateBrandRequest
	if err := json.Unmarshal(badan, &req); err != nil {
		t.Fatal(err)
	}
	if err := pertahankanNominalKosong(badan, &req, lama); err != nil {
		t.Fatal(err)
	}
	if req.UjrohClosing != 0 {
		t.Errorf("ujroh_closing dikirim 0 harus tetap 0, dapat %v", req.UjrohClosing)
	}
	if req.MinimalDP != 5_000_000 || req.CashbackAlumni != 250_000 || req.BiayaPendaftaranAgen != 300_000 || req.UjrohPembinaan != 100_000 || req.UjrohRepeatBooking != 50_000 {
		t.Errorf("nominal yang tidak dikirim harus mengikuti nilai tersimpan: %+v", req)
	}

	// Semua dikirim: tidak ada yang ditimpa.
	lengkap := []byte(`{"minimal_dp":1,"cashback_alumni":2,"biaya_pendaftaran_agen":3,"ujroh_closing":4,"ujroh_pembinaan":5,"ujroh_repeat_booking":6}`)
	var req2 UpdateBrandRequest
	if err := json.Unmarshal(lengkap, &req2); err != nil {
		t.Fatal(err)
	}
	if err := pertahankanNominalKosong(lengkap, &req2, lama); err != nil {
		t.Fatal(err)
	}
	if req2.MinimalDP != 1 || req2.CashbackAlumni != 2 || req2.BiayaPendaftaranAgen != 3 || req2.UjrohClosing != 4 || req2.UjrohPembinaan != 5 || req2.UjrohRepeatBooking != 6 {
		t.Errorf("nominal yang dikirim tidak boleh ditimpa: %+v", req2)
	}
}
