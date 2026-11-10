package shared

import (
	"strings"
	"time"
)

// RupiahBulat: nominal rupiah harus bilangan bulat. Nominal disimpan sebagai float64, jadi pecahan
// ditolak di pintu masuk.
func RupiahBulat(v float64) bool {
	return v == float64(int64(v))
}

// TanggalTidakDiMasaDepan memeriksa teks tanggal YYYY-MM-DD: harus tanggal yang sah dan tidak lebih
// dari hari ini (WIB). Dipakai untuk tanggal pembayaran dan pengembalian dana, yang masuk laporan.
func TanggalTidakDiMasaDepan(s string) bool {
	s = strings.TrimSpace(s)
	tgl, err := time.Parse("2006-01-02", s)
	if err != nil {
		return false
	}
	return tgl.Format("2006-01-02") <= TodayWIB()
}
