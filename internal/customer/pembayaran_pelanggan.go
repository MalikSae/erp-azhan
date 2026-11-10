package customer

import "erp-azhan/api/internal/payment"

// pembayaranUntukPelanggan menyiapkan riwayat pembayaran untuk portal pelanggan. Pembayaran yang dibatalkan
// admin (status void) ditampilkan sebagai ditolak dengan keterangan umum: portal hanya mengenal pending,
// confirmed, dan rejected, dan alasan internal admin tidak ikut dikirim.
func pembayaranUntukPelanggan(items []payment.Payment) []payment.Payment {
	out := make([]payment.Payment, 0, len(items))
	for _, p := range items {
		if p.Status == "void" {
			keterangan := "Pembayaran ini dibatalkan oleh admin. Hubungi admin untuk keterangan lebih lanjut."
			p.Status = "rejected"
			p.RejectionReason = &keterangan
		}
		p.VoidReason, p.VoidedBy, p.VoidedAt = nil, nil, nil
		out = append(out, p)
	}
	return out
}
