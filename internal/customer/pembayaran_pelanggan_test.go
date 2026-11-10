package customer

import (
	"testing"

	"erp-azhan/api/internal/payment"
)

// T-03: pembayaran void tampil ke pelanggan sebagai ditolak, tanpa alasan internal admin.
func TestPembayaranUntukPelanggan(t *testing.T) {
	alasan := "salah booking, seharusnya ASAB12"
	pelaku := int64(7)
	out := pembayaranUntukPelanggan([]payment.Payment{
		{ID: 1, Status: "confirmed"},
		{ID: 2, Status: "void", VoidReason: &alasan, VoidedBy: &pelaku},
	})
	if len(out) != 2 || out[0].Status != "confirmed" || out[0].RejectionReason != nil {
		t.Fatalf("pembayaran biasa berubah: %+v", out)
	}
	v := out[1]
	if v.Status != "rejected" || v.RejectionReason == nil || v.VoidReason != nil || v.VoidedBy != nil {
		t.Fatalf("void = %+v", v)
	}
	if got := pembayaranUntukPelanggan(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil harus menjadi daftar kosong: %#v", got)
	}
}
