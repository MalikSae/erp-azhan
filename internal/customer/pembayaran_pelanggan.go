package customer

import (
	"strconv"
	"strings"

	"erp-azhan/api/internal/payment"
)

// metodeTransfer: satu-satunya metode pembayaran yang bisa dikirim dari portal.
const metodeTransfer = "transfer"

// Batas panjang isian pengirim, mengikuti lebar kolom payments.sender_name dan payments.sender_bank.
const (
	maksNamaPengirim = 150
	maksBankPengirim = 100
)

// rapikanPengirim memangkas nama dan bank pengirim lalu memeriksa panjangnya. Mengembalikan pesan
// kesalahan untuk pengguna, atau teks kosong bila sah.
func rapikanPengirim(req *payment.CreatePaymentRequest) string {
	nama := strings.TrimSpace(*req.SenderName)
	if len([]rune(nama)) > maksNamaPengirim {
		return "nama pengirim terlalu panjang"
	}
	req.SenderName = &nama
	if req.SenderBank != nil {
		bank := strings.TrimSpace(*req.SenderBank)
		if len([]rune(bank)) > maksBankPengirim {
			return "nama bank pengirim terlalu panjang"
		}
		req.SenderBank = &bank
		if bank == "" {
			req.SenderBank = nil
		}
	}
	return ""
}

// pesanMelebihiMenunggu menolak konfirmasi baru bila bersama konfirmasi yang masih menunggu verifikasi
// jumlahnya melebihi sisa tagihan. Mengembalikan teks kosong bila masih muat.
func pesanMelebihiMenunggu(jumlah, menunggu, sisa float64) string {
	if menunggu > 0 && jumlah+menunggu > sisa {
		return "sudah ada konfirmasi pembayaran yang menunggu verifikasi. Nominal baru bersama yang menunggu melebihi sisa tagihan"
	}
	return ""
}

// pesanKurangDP: selama pembayaran terkonfirmasi ditambah konfirmasi yang menunggu verifikasi masih di bawah
// minimal DP, konfirmasi dari portal wajib menutup kekurangan DP sekaligus (keputusan 11 November 2026).
// Cicilan DP hanya dicatat admin. Mengembalikan teks kosong bila nominal memenuhi.
func pesanKurangDP(jumlah, minimalDP, terbayar, menunggu float64) string {
	kurang := minimalDP - terbayar - menunggu
	if kurang <= 0 || jumlah >= kurang {
		return ""
	}
	return "nominal minimal " + rupiahTeks(kurang) + " untuk memenuhi DP. Hubungi admin bila ingin mencicil DP"
}

// rupiahTeks menulis nominal bulat dengan titik ribuan, misalnya Rp5.000.000.
func rupiahTeks(v float64) string {
	digit := strconv.FormatInt(int64(v), 10)
	var b strings.Builder
	b.WriteString("Rp")
	for i, c := range digit {
		if i > 0 && (len(digit)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// pembayaranUntukPelanggan menyiapkan riwayat pembayaran untuk portal pelanggan. Pembayaran yang dibatalkan
// admin (status void) ditampilkan sebagai ditolak dengan keterangan umum: portal hanya mengenal pending,
// confirmed, dan rejected, dan alasan internal admin tidak ikut dikirim. Kolom kerja admin (catatan,
// penerima tunai, id admin yang memverifikasi) juga tidak dikirim.
func pembayaranUntukPelanggan(items []payment.Payment) []payment.Payment {
	out := make([]payment.Payment, 0, len(items))
	for _, p := range items {
		if p.Status == payment.StatusVoid {
			keterangan := "Pembayaran ini dibatalkan oleh admin. Hubungi admin untuk keterangan lebih lanjut."
			p.Status = payment.StatusRejected
			p.RejectionReason = &keterangan
		}
		p.VoidReason, p.VoidedBy, p.VoidedAt = nil, nil, nil
		p.Notes, p.ReceivedBy, p.VerifiedBy, p.DicatatOleh = nil, nil, nil, nil
		out = append(out, p)
	}
	return out
}
