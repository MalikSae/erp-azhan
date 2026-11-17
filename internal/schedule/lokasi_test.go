package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

const fotoLokasiUji = "/uploads/haramain/11111111-2222-4333-8444-000000000009.webp"

func tempatUji(t *testing.T, tx *sql.Tx, nama, kota string, aktif bool, urutan int) int64 {
	t.Helper()
	res, err := tx.Exec(`INSERT INTO haramain_tempat (nama, kategori, kota, foto_url, ringkas, aktif, urutan)
		VALUES (?, 'masjid', ?, ?, 'Uji', ?, ?)`, nama, kota, fotoLokasiUji, aktif, urutan)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func pesanLokasi(err error) string {
	var le LokasiError
	if errors.As(err, &le) {
		return le.Error()
	}
	return ""
}

func TestCekDaftarLokasi(t *testing.T) {
	banyak := make([]int64, MaksLokasi+1)
	for i := range banyak {
		banyak[i] = int64(i + 1)
	}
	for nama, ids := range map[string][]int64{
		"lebih dari 12": banyak,
		"dobel":         {3, 5, 3},
		"id nol":        {0},
	} {
		if pesanLokasi(cekDaftarLokasi(ids)) == "" {
			t.Errorf("%s: harus ditolak", nama)
		}
	}
	if err := cekDaftarLokasi(banyak[:MaksLokasi]); err != nil {
		t.Errorf("12 lokasi harus lolos: %v", err)
	}
	if err := cekDaftarLokasi(nil); err != nil {
		t.Errorf("kosong harus lolos: %v", err)
	}
}

// Absent lokasi_ids must decode as nil (keep stored), [] as an empty list (clear).
func TestLokasiIDsMembedakanTidakDikirimDanKosong(t *testing.T) {
	var tanpa, kosong, nol, isi UpdateScheduleRequest
	_ = json.Unmarshal([]byte(`{"jadwal_nama":"x"}`), &tanpa)
	_ = json.Unmarshal([]byte(`{"lokasi_ids":[]}`), &kosong)
	_ = json.Unmarshal([]byte(`{"lokasi_ids":null}`), &nol)
	_ = json.Unmarshal([]byte(`{"lokasi_ids":[7,2,9]}`), &isi)
	if tanpa.LokasiIDs != nil || nol.LokasiIDs != nil {
		t.Fatal("kunci tidak dikirim atau null harus nil")
	}
	if kosong.LokasiIDs == nil || len(*kosong.LokasiIDs) != 0 {
		t.Fatal("daftar kosong harus terbaca daftar kosong")
	}
	if isi.LokasiIDs == nil || !reflect.DeepEqual(*isi.LokasiIDs, []int64{7, 2, 9}) {
		t.Fatalf("urutan harus dipertahankan: %v", isi.LokasiIDs)
	}
}

// Public package always sends "lokasi" as an array.
func TestPackageV1LokasiKosong(t *testing.T) {
	raw, _ := json.Marshal(newPackageV1(&PublicSchedule{}, Klasifikasi{JourneyType: JourneyUmroh}, "2026-11-19"))
	if !strings.Contains(string(raw), `"lokasi":[]`) {
		t.Fatalf("paket tanpa lokasi harus mengirim array kosong: %s", raw)
	}
}

func TestLokasiPaket(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, _ := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	testdb.Exec(t, tx, `DELETE FROM schedule_lokasi WHERE schedule_id = ?`, sched)

	a := tempatUji(t, tx, "Lokasi A Qzp", "makkah", true, 90001)
	b := tempatUji(t, tx, "Lokasi B Qzp", "madinah", true, 90002)
	mati := tempatUji(t, tx, "Lokasi Mati Qzp", "makkah", false, 90003)

	ids := func() []int64 {
		t.Helper()
		got, err := lokasiIDsJadwal(ctx, tx, sched)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	// Empty paket: [] not nil.
	if got := ids(); got == nil || len(got) != 0 {
		t.Fatalf("paket tanpa lokasi harus []: %v", got)
	}

	// Save keeps admin order.
	if err := gantiLokasi(ctx, tx, sched, []int64{b, mati, a}); err != nil {
		t.Fatal(err)
	}
	if got := ids(); !reflect.DeepEqual(got, []int64{b, mati, a}) {
		t.Fatalf("urutan simpan salah: %v", got)
	}

	// Admin detail: inactive kept with its name and flag.
	adm, err := lokasiJadwal(ctx, tx, sched)
	if err != nil {
		t.Fatal(err)
	}
	if len(adm) != 3 || adm[1].Nama != "Lokasi Mati Qzp" || adm[1].Aktif || !adm[0].Aktif || adm[0].Kota != "madinah" || adm[0].FotoURL != fotoLokasiUji {
		t.Fatalf("lokasi admin salah: %+v", adm)
	}

	// Public: inactive hidden, order kept.
	pub, err := lokasiPublik(ctx, tx, []int64{sched, 0})
	if err != nil {
		t.Fatal(err)
	}
	if l := pub[sched]; len(l) != 2 || l[0].ID != b || l[1].ID != a || l[0].Nama != "Lokasi B Qzp" || l[0].Kota != "madinah" || l[0].FotoURL != fotoLokasiUji {
		t.Fatalf("lokasi publik salah: %+v", l)
	}
	if _, ada := pub[0]; ada {
		t.Fatal("paket tanpa lokasi tidak boleh punya kunci")
	}

	// Replace.
	if err := gantiLokasi(ctx, tx, sched, []int64{a}); err != nil {
		t.Fatal(err)
	}
	if got := ids(); !reflect.DeepEqual(got, []int64{a}) {
		t.Fatalf("ganti lokasi salah: %v", got)
	}

	// Clear.
	if err := gantiLokasi(ctx, tx, sched, []int64{}); err != nil {
		t.Fatal(err)
	}
	if got := ids(); len(got) != 0 {
		t.Fatalf("hapus lokasi salah: %v", got)
	}

	// Validation against the master: unknown id rejected, inactive allowed (stays linked).
	if pesanLokasi(validasiLokasi(ctx, tx, []int64{a, 999999999})) == "" {
		t.Error("id tidak ada harus ditolak")
	}
	if pesanLokasi(validasiLokasi(ctx, tx, []int64{a, a})) == "" {
		t.Error("dobel harus ditolak")
	}
	if err := validasiLokasi(ctx, tx, []int64{a, mati, b}); err != nil {
		t.Errorf("lokasi sah ditolak: %v", err)
	}

	// Options: active only, master order.
	opsi, err := opsiLokasi(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	posA, posB := -1, -1
	for i, o := range opsi {
		switch o.ID {
		case mati:
			t.Error("tempat nonaktif tidak boleh jadi pilihan")
		case a:
			posA = i
		case b:
			posB = i
		}
	}
	if posA < 0 || posB < 0 || posA > posB {
		t.Fatalf("urutan pilihan salah: a=%d b=%d", posA, posB)
	}
}
