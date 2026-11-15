package haramain

import (
	"context"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

const fotoUji = "/uploads/haramain/11111111-2222-4333-8444-000000000001.webp"
const fotoUlasanUji = "/uploads/ulasan-haramain/11111111-2222-4333-8444-000000000002.jpg"

func inputSah() Input {
	return Input{Nama: "Masjid Uji Qzp", Kategori: "masjid", Kota: "madinah", FotoURL: fotoUji, Ringkas: "Masjid", Aktif: true,
		Galeri: []string{fotoUji}}
}

func galatValidasi(err error) bool {
	var v *ErrValidasi
	return errors.As(err, &v)
}

func TestValidasiTempat(t *testing.T) {
	for nama, ubah := range map[string]func(*Input){
		"nama kosong":      func(i *Input) { i.Nama = " " },
		"kategori salah":   func(i *Input) { i.Kategori = "hotel" },
		"kota salah":       func(i *Input) { i.Kota = "jeddah" },
		"foto luar":        func(i *Input) { i.FotoURL = "/uploads/brand-logos/x.png" },
		"galeri luar":      func(i *Input) { i.Galeri = []string{"https://contoh.id/a.jpg"} },
		"maps bukan https": func(i *Input) { s := "http://maps.google.com"; i.GmapsURL = &s },
	} {
		in := inputSah()
		ubah(&in)
		if err := in.validasi(); !galatValidasi(err) {
			t.Errorf("%s: harus ditolak (%v)", nama, err)
		}
	}
	in := inputSah()
	if err := in.validasi(); err != nil {
		t.Errorf("input sah ditolak: %v", err)
	}
}

func TestTempatDanUlasan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	now := time.Date(2099, 4, 1, 10, 0, 0, 0, ZonaWIB)

	id, err := Buat(ctx, tx, 0, inputSah())
	if err != nil {
		t.Fatal(err)
	}
	nonaktif := inputSah()
	nonaktif.Nama, nonaktif.Aktif = "Tempat Nonaktif Qzp", false
	idMati, err := Buat(ctx, tx, 0, nonaktif)
	if err != nil {
		t.Fatal(err)
	}
	publik, _ := Daftar(ctx, tx, true)
	for _, p := range publik {
		if p.ID == idMati {
			t.Error("tempat nonaktif tampil di publik")
		}
	}
	if _, err := Ambil(ctx, tx, idMati, true); !errors.Is(err, ErrTidakAda) {
		t.Errorf("detail nonaktif publik: %v", err)
	}

	penulis := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PenulisUlasanQzp"})
	pelapor := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PelaporUlasanQzp"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahUlasanQzp"})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, penulis, b)

	// Belum lunas: belum boleh menulis.
	if err := SimpanUlasan(ctx, tx, id, penulis, InputUlasan{Teks: "Bagus"}, now); !errors.Is(err, ErrBelumBerangkat) {
		t.Errorf("booking dp: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status = 'lunas' WHERE id = ?`, b)
	// Lunas tetapi belum berangkat.
	if err := SimpanUlasan(ctx, tx, id, penulis, InputUlasan{Teks: "Bagus"}, time.Date(2099, 2, 1, 0, 0, 0, 0, ZonaWIB)); !errors.Is(err, ErrBelumBerangkat) {
		t.Errorf("belum berangkat: %v", err)
	}
	if err := SimpanUlasan(ctx, tx, id, penulis, InputUlasan{Teks: " "}, now); !galatValidasi(err) {
		t.Errorf("teks kosong: %v", err)
	}
	if err := SimpanUlasan(ctx, tx, id, penulis, InputUlasan{Teks: "x", Foto: []string{fotoUji}}, now); !galatValidasi(err) {
		t.Errorf("foto bukan kategori ulasan: %v", err)
	}
	if err := SimpanUlasan(ctx, tx, id, penulis, InputUlasan{Teks: "Masuk dari pintu 25 lebih lega.", Foto: []string{fotoUlasanUji}}, now); err != nil {
		t.Fatal(err)
	}
	// Simpan kedua memperbarui ulasan yang sama.
	if err := SimpanUlasan(ctx, tx, id, penulis, InputUlasan{Teks: "Diperbarui."}, now); err != nil {
		t.Fatal(err)
	}
	saya, err := UlasanSaya(ctx, tx, id, penulis)
	if err != nil || saya == nil || saya.Teks != "Diperbarui." || len(saya.Foto) != 0 {
		t.Fatalf("ulasan saya: %+v err=%v", saya, err)
	}
	if tm, _ := Ambil(ctx, tx, id, true); tm.JumlahUlasan != 1 {
		t.Errorf("jumlah ulasan: %d", tm.JumlahUlasan)
	}

	// Laporan: tidak bisa melaporkan sendiri; satu kali per akun.
	if err := Laporkan(ctx, tx, saya.ID, penulis); !galatValidasi(err) {
		t.Errorf("lapor sendiri: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := Laporkan(ctx, tx, saya.ID, pelapor); err != nil {
			t.Fatal(err)
		}
	}
	dilaporkan, _ := UlasanAdmin(ctx, tx, true)
	var ketemu *Ulasan
	for i := range dilaporkan {
		if dilaporkan[i].ID == saya.ID {
			ketemu = &dilaporkan[i]
		}
	}
	if ketemu == nil || ketemu.JumlahLaporan != 1 {
		t.Fatalf("tab dilaporkan: %+v", ketemu)
	}
	if err := Abaikan(ctx, tx, saya.ID); err != nil {
		t.Fatal(err)
	}
	dilaporkan, _ = UlasanAdmin(ctx, tx, true)
	for _, u := range dilaporkan {
		if u.ID == saya.ID {
			t.Error("ulasan masih di tab dilaporkan setelah diabaikan")
		}
	}

	// Disembunyikan: hilang dari publik dan dari hitungan.
	if err := AturTampil(ctx, tx, saya.ID, 0, true); err != nil {
		t.Fatal(err)
	}
	if u, _ := UlasanTempat(ctx, tx, id); len(u) != 0 {
		t.Errorf("ulasan tersembunyi tampil: %+v", u)
	}
	if tm, _ := Ambil(ctx, tx, id, true); tm.JumlahUlasan != 0 {
		t.Errorf("jumlah setelah disembunyikan: %d", tm.JumlahUlasan)
	}

	if err := HapusUlasan(ctx, tx, id, penulis); err != nil {
		t.Fatal(err)
	}
	if err := HapusUlasan(ctx, tx, id, penulis); !errors.Is(err, ErrUlasanTidakAda) {
		t.Errorf("hapus kedua: %v", err)
	}
	if err := Hapus(ctx, tx, idMati); err != nil {
		t.Fatal(err)
	}
}
