// Package benefit mengelola master "Benefit Paket": highlight benefit (ikon + teks pendek) per brand yang
// dipilih tiap paket (keputusan 13 November 2026).
package benefit

import (
	"errors"
	"strings"
	"time"
)

const (
	// MaksTeks: panjang teks benefit dalam karakter (rune), sama dengan lebar kolom paket_benefits.teks.
	MaksTeks = 30
	// JumlahPerPaket: paket yang dijual wajib punya tepat sebanyak ini; paket draft boleh kurang.
	JumlahPerPaket = 4
)

// Ikon adalah satu pilihan ikon benefit: kunci ikon dan label Indonesianya.
type Ikon struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// DaftarIkon adalah daftar tetap ikon benefit, berurutan. Dashboard mengambilnya lewat API supaya label
// tidak ditulis ulang di sisi klien.
var DaftarIkon = []Ikon{
	{"solar:bed-bold-duotone", "Hotel"},
	{"solar:buildings-2-bold-duotone", "Hotel berbintang"},
	{"solar:plane-2-bold-duotone", "Penerbangan"},
	{"solar:ticket-bold-duotone", "Tiket"},
	{"solar:bus-bold-duotone", "Bus"},
	{"solar:tram-bold-duotone", "Kereta cepat"},
	{"solar:route-bold-duotone", "City tour"},
	{"solar:map-point-bold-duotone", "Ziarah"},
	{"solar:chef-hat-bold-duotone", "Makan"},
	{"solar:cup-hot-bold-duotone", "Minuman & snack"},
	{"solar:bottle-bold-duotone", "Air zamzam"},
	{"solar:user-speak-bold-duotone", "Pembimbing"},
	{"solar:users-group-rounded-bold-duotone", "Rombongan"},
	{"solar:user-hand-up-bold-duotone", "Tour leader"},
	{"solar:headphones-round-bold-duotone", "Layanan 24 jam"},
	{"solar:suitcase-tag-bold-duotone", "Koper"},
	{"solar:bag-3-bold-duotone", "Perlengkapan"},
	{"solar:t-shirt-bold-duotone", "Seragam"},
	{"solar:passport-bold-duotone", "Visa"},
	{"solar:document-text-bold-duotone", "Dokumen"},
	{"solar:shield-check-bold-duotone", "Asuransi"},
	{"solar:medical-kit-bold-duotone", "Kesehatan"},
	{"solar:accessibility-bold-duotone", "Ramah lansia"},
	{"solar:wi-fi-router-bold-duotone", "Internet"},
	{"solar:phone-bold-duotone", "Kartu SIM"},
	{"solar:camera-bold-duotone", "Dokumentasi"},
	{"solar:calendar-bold-duotone", "Jadwal"},
	{"solar:clock-circle-bold-duotone", "Durasi"},
	{"solar:tag-price-bold-duotone", "Harga"},
	{"solar:gift-bold-duotone", "Bonus"},
	{"solar:star-bold-duotone", "Unggulan"},
	{"solar:crown-bold-duotone", "VIP"},
	{"solar:verified-check-bold-duotone", "Resmi"},
	{"solar:hand-heart-bold-duotone", "Manasik"},
	{"solar:moon-stars-bold-duotone", "Ramadhan"},
}

// IkonSah memeriksa kunci ikon ada di DaftarIkon.
func IkonSah(key string) bool {
	for _, i := range DaftarIkon {
		if i.Key == key {
			return true
		}
	}
	return false
}

// ValidasiIsi merapikan dan memeriksa teks serta ikon benefit. Mengembalikan nilai yang sudah dirapikan,
// atau error berisi pesan untuk pengguna (400).
func ValidasiIsi(teks, ikon string) (string, string, error) {
	teks = strings.TrimSpace(teks)
	ikon = strings.TrimSpace(ikon)
	if teks == "" {
		return "", "", errors.New("teks benefit wajib diisi")
	}
	if len([]rune(teks)) > MaksTeks {
		return "", "", errors.New("teks benefit maksimal 30 karakter")
	}
	if ikon == "" {
		return "", "", errors.New("ikon benefit wajib dipilih")
	}
	if !IkonSah(ikon) {
		return "", "", errors.New("ikon benefit tidak ada di daftar ikon")
	}
	return teks, ikon, nil
}

// Benefit merepresentasikan satu baris master benefit paket.
type Benefit struct {
	ID       int64  `json:"id"`
	BrandID  int64  `json:"brand_id"`
	Teks     string `json:"teks"`
	Ikon     string `json:"ikon"`
	IsActive bool   `json:"is_active"`
	// Dipakai: jumlah paket yang memakai benefit ini. Benefit yang masih dipakai tidak bisa dihapus.
	Dipakai   int       `json:"dipakai"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Ref adalah benefit pada response admin sebuah paket.
type Ref struct {
	ID       int64  `json:"id"`
	Teks     string `json:"teks"`
	Ikon     string `json:"ikon"`
	IsActive bool   `json:"is_active"`
}

// RefPublik adalah benefit pada response publik sebuah paket.
type RefPublik struct {
	Teks string `json:"teks"`
	Ikon string `json:"ikon"`
}

// KePublik mengubah daftar benefit admin menjadi bentuk publik; hasilnya tidak pernah nil.
func KePublik(refs []Ref) []RefPublik {
	out := make([]RefPublik, 0, len(refs))
	for _, r := range refs {
		out = append(out, RefPublik{Teks: r.Teks, Ikon: r.Ikon})
	}
	return out
}

// CreateRequest payload untuk membuat benefit. BrandID hanya dipakai admin holding.
type CreateRequest struct {
	BrandID *int64 `json:"brand_id"`
	Teks    string `json:"teks"`
	Ikon    string `json:"ikon"`
}

// UpdateRequest payload untuk mengubah benefit. IsActive nil berarti status aktif tidak diubah.
type UpdateRequest struct {
	Teks     string `json:"teks"`
	Ikon     string `json:"ikon"`
	IsActive *bool  `json:"is_active"`
}

// Opsi adalah jawaban GET /api/admin/schedules/benefit-options untuk form paket.
type Opsi struct {
	Ikon     []Ikon    `json:"ikon"`
	Benefits []Benefit `json:"benefits"`
}

// PilihanError: daftar benefit yang dipilih sebuah paket tidak sah; pesannya ditampilkan ke pengguna (400).
type PilihanError string

func (e PilihanError) Error() string { return string(e) }

// ValidasiDaftarID memeriksa bentuk daftar benefit sebuah paket tanpa menyentuh database: paling banyak
// empat, id positif, dan tidak ada yang kembar.
func ValidasiDaftarID(ids []int64) error {
	if len(ids) > JumlahPerPaket {
		return PilihanError("highlight benefit paling banyak 4")
	}
	lihat := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return PilihanError("benefit_ids berisi id yang tidak valid")
		}
		if lihat[id] {
			return PilihanError("highlight benefit tidak boleh kembar")
		}
		lihat[id] = true
	}
	return nil
}
