package pengumuman

import (
	"context"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestValidasiInput(t *testing.T) {
	ok := Input{Judul: "Jadwal manasik", Isi: "Manasik hari Sabtu.", Mulai: "2026-11-01", Akhir: "2026-11-30"}
	if msg := ok.validasi(); msg != "" {
		t.Fatalf("input sah ditolak: %s", msg)
	}
	for nama, ubah := range map[string]func(*Input){
		"judul pendek":       func(i *Input) { i.Judul = "ab" },
		"isi kosong":         func(i *Input) { i.Isi = "  " },
		"tanggal salah":      func(i *Input) { i.Mulai = "01-11-2026" },
		"akhir sebelum awal": func(i *Input) { i.Akhir = "2026-10-31" },
	} {
		in := ok
		ubah(&in)
		if in.validasi() == "" {
			t.Errorf("%s: harus ditolak", nama)
		}
	}
}

func TestStatusTayang(t *testing.T) {
	for hari, want := range map[string]string{"2026-10-31": "akan_tayang", "2026-11-01": "tayang", "2026-11-30": "tayang", "2026-12-01": "berakhir"} {
		if got := status("2026-11-01", "2026-11-30", hari); got != want {
			t.Errorf("status(%s) = %s, want %s", hari, got, want)
		}
	}
}

func TestPengumumanUntukAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	punya := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PunyaBookingQzp"})
	tanpa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TanpaBookingQzp"})
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahQzp"})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: jamaah})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, punya, b)

	sekarang := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	umum, err := Buat(ctx, tx, brand, 0, Input{Judul: "Update perlengkapan ihram", Isi: "Daftar terbaru tersedia.", Mulai: "2026-11-01", Akhir: "2026-11-30"})
	if err != nil {
		t.Fatal(err)
	}
	khusus, err := Buat(ctx, tx, brand, 0, Input{ScheduleID: &jadwal, Judul: "Jadwal manasik kelompok", Isi: "Manasik hari Sabtu.", Mulai: "2026-11-05", Akhir: "2026-11-20"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Buat(ctx, tx, brand, 0, Input{Judul: "Sudah berakhir", Isi: "Lama.", Mulai: "2026-10-01", Akhir: "2026-10-31"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Buat(ctx, tx, brand, 0, Input{Judul: "Belum tayang", Isi: "Nanti.", Mulai: "2026-12-01", Akhir: "2026-12-31"}); err != nil {
		t.Fatal(err)
	}

	items, total, err := UntukAkun(ctx, tx, brand, punya, 0, sekarang)
	if err != nil || total != 2 || len(items) != 2 {
		t.Fatalf("akun dengan booking: total=%d items=%d err=%v", total, len(items), err)
	}
	if items[0].ID != khusus || items[1].ID != umum { // terbaru (mulai tayang) di atas
		t.Errorf("urutan = %d, %d; want %d, %d", items[0].ID, items[1].ID, khusus, umum)
	}
	if items[0].JadwalNama == "" || items[0].Status != "tayang" {
		t.Errorf("pengumuman khusus jadwal: %+v", items[0])
	}

	// limit membatasi tampilan tetapi total tetap jumlah seluruhnya.
	satu, total, _ := UntukAkun(ctx, tx, brand, punya, 1, sekarang)
	if len(satu) != 1 || total != 2 {
		t.Errorf("limit 1: items=%d total=%d", len(satu), total)
	}

	// Akun tanpa booking di jadwal itu hanya melihat yang untuk semua.
	items, total, _ = UntukAkun(ctx, tx, brand, tanpa, 0, sekarang)
	if total != 1 || len(items) != 1 || items[0].ID != umum {
		t.Errorf("akun tanpa booking: %+v total=%d", items, total)
	}

	// Booking batal tidak membuka pengumuman jadwalnya.
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	if _, total, _ = UntukAkun(ctx, tx, brand, punya, 0, sekarang); total != 1 {
		t.Errorf("booking batal: total=%d, want 1", total)
	}
}

func TestPengumumanAdminScopeDanJadwal(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	lain := brand + 99999

	id, err := Buat(ctx, tx, brand, 0, Input{Judul: "Pengumuman uji", Isi: "Isi uji.", Mulai: "2026-11-01", Akhir: "2026-11-30"})
	if err != nil {
		t.Fatal(err)
	}
	// Jadwal brand lain ditolak.
	if _, err := Buat(ctx, tx, lain, 0, Input{ScheduleID: &jadwal, Judul: "Salah brand", Isi: "Isi.", Mulai: "2026-11-01", Akhir: "2026-11-30"}); err == nil {
		t.Error("jadwal milik brand lain harus ditolak")
	}
	// Admin brand lain tidak bisa mengubah atau menghapus.
	if err := Ubah(ctx, tx, id, &lain, Input{Judul: "Diubah", Isi: "Isi.", Mulai: "2026-11-01", Akhir: "2026-11-30"}); !errors.Is(err, ErrTidakAda) {
		t.Errorf("ubah lintas brand: %v", err)
	}
	if err := Hapus(ctx, tx, id, &lain); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus lintas brand: %v", err)
	}
	if err := Ubah(ctx, tx, id, &brand, Input{ScheduleID: &jadwal, Judul: "Diubah", Isi: "Isi baru.", Mulai: "2026-11-02", Akhir: "2026-11-29"}); err != nil {
		t.Fatal(err)
	}
	daftar, err := Daftar(ctx, tx, &brand, time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var ketemu bool
	for _, p := range daftar {
		if p.ID == id {
			ketemu = p.Judul == "Diubah" && p.ScheduleID != nil && p.Status == "tayang"
		}
	}
	if !ketemu {
		t.Errorf("pengumuman tidak tampil sesuai di daftar admin: %+v", daftar)
	}
	if err := Hapus(ctx, tx, id, &brand); err != nil {
		t.Fatal(err)
	}
}
