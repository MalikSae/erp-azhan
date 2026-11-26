// Package tabungan: Tabungan Umroh di app (keputusan 17 November 2026).
//
// Catatan target menabung pribadi calon jamaah. Hana tidak menyimpan uang: paket ini hanya
// mencatat target dan angka yang diisi sendiri oleh akun. Uang selalu bilangan bulat rupiah.
// Hari ini dan bulan berjalan memakai zona WIB. Token push disimpan di sini tetapi belum dipakai
// mengirim apa pun pada tahap ini.
package tabungan

import "time"

// ZonaWIB: zona waktu untuk hari ini dan bulan berjalan.
var ZonaWIB = time.FixedZone("WIB", 7*3600)

// Batas kontrak.
const (
	MinTarget         = 1_000_000
	MaksTarget        = 2_000_000_000
	MinCatatan        = 1_000
	MaksCatatan       = 1_000_000_000
	MaksBulanKeDepan  = 36
	MaksNamaPaket     = 150
	MaksJamaah        = 9
	MaksTokenPerAkun  = 10
	MaksDaftarCatatan = 200
	KelipatanPerBulan = 1000
)

// TanggalAwalCatatan: catatan tidak boleh sebelum tanggal ini.
var TanggalAwalCatatan = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// TipeKamar mengikuti nilai yang dipakai booking.
var TipeKamar = map[string]bool{"Quad": true, "Triple": true, "Double": true}

// Target: satu target per akun. Dipakai sebagai isi PUT dan bagian GET.
type Target struct {
	Jenis            string `json:"jenis"`
	ScheduleID       *int64 `json:"schedule_id"`
	NamaPaket        string `json:"nama_paket"`
	BulanBerangkat   string `json:"bulan_berangkat"`
	JumlahDewasa     int    `json:"jumlah_dewasa"`
	JumlahBayi       int    `json:"jumlah_bayi"`
	TipeKamar        string `json:"tipe_kamar"`
	TargetTotal      int64  `json:"target_total"`
	DPMinimal        int64  `json:"dp_minimal"`
	PengingatAktif   bool   `json:"pengingat_aktif"`
	PengingatTanggal int    `json:"pengingat_tanggal"`
	PengingatJam     int    `json:"pengingat_jam"`
}

// TargetOut: target beserta waktu dibuat.
type TargetOut struct {
	Target
	DibuatAt string `json:"dibuat_at"`
}

// Ringkasan: angka turunan dari target dan catatan.
type Ringkasan struct {
	Tercatat         int64   `json:"tercatat"`
	Sisa             int64   `json:"sisa"`
	Persen           int     `json:"persen"`
	SisaBulan        int     `json:"sisa_bulan"`
	PerBulan         int64   `json:"per_bulan"`
	TercatatBulanIni int64   `json:"tercatat_bulan_ini"`
	PerBulanIni      int64   `json:"per_bulan_ini"`
	KurangBulanIni   int64   `json:"kurang_bulan_ini"`
	DPTercapai       bool    `json:"dp_tercapai"`
	Tercapai         bool    `json:"tercapai"`
	Lewat            bool    `json:"lewat"`
	PaketTayang      bool    `json:"paket_tayang"`
	BatasPelunasan   *string `json:"batas_pelunasan"`
}

// Jadwal: jadwal (paket) yang dipakai target jenis paket. Ada false bila target perkiraan
// atau jadwalnya sudah dihapus.
type Jadwal struct {
	Ada                bool
	BerangkatTanggal   string // YYYY-MM-DD
	Status             string
	BatasPelunasanHari int
}

// Tayang: berstatus published dan belum berangkat (tanggal WIB).
func (j Jadwal) Tayang(now time.Time) bool {
	return j.Ada && j.Status == "published" && j.BerangkatTanggal >= now.In(ZonaWIB).Format("2006-01-02")
}

// Catatan: satu catatan tabungan.
type Catatan struct {
	ID       int64  `json:"id"`
	Jumlah   int64  `json:"jumlah"`
	Tanggal  string `json:"tanggal"`
	DibuatAt string `json:"dibuat_at"`
}

// Respons: bentuk GET dan balasan semua perubahan.
type Respons struct {
	Target    *TargetOut `json:"target"`
	Ringkasan *Ringkasan `json:"ringkasan"`
	Catatan   []Catatan  `json:"catatan"`
}
