package mailer

import (
	"fmt"
	"strings"
)

func VerifyEmail(brandName, nama, link string) (subject, text string) {
	subject = "Verifikasi email akun " + brandName
	text = fmt.Sprintf(`Halo %s,

Terima kasih sudah mendaftar akun di %s. Buka tautan berikut untuk memverifikasi email Anda (berlaku 24 jam):

%s

Jika Anda tidak merasa mendaftar, abaikan email ini.

%s`, nama, brandName, link, brandName)
	return
}

func ResetPassword(brandName, nama, link string) (subject, text string) {
	subject = "Atur ulang password " + brandName
	text = fmt.Sprintf(`Halo %s,

Kami menerima permintaan mengatur ulang password akun Anda di %s. Buka tautan berikut untuk membuat password baru (berlaku 1 jam):

%s

Jika Anda tidak memintanya, abaikan email ini. Password Anda tidak berubah.

%s`, nama, brandName, link, brandName)
	return
}

type BankInfo struct{ Bank, Nomor, AtasNama string }

func OrderCreated(brandName, nama, kodeBooking, paket, tglBerangkat, total, dp, batasBayar, link string, banks []BankInfo) (subject, text string) {
	subject = fmt.Sprintf("Booking %s berhasil dibuat (%s)", paket, kodeBooking)
	var b strings.Builder
	for _, k := range banks {
		fmt.Fprintf(&b, "  - %s %s a.n. %s\n", k.Bank, k.Nomor, k.AtasNama)
	}
	if b.Len() == 0 {
		b.WriteString("  (hubungi kami untuk rekening tujuan)\n")
	}
	text = fmt.Sprintf(`Halo %s,

Booking seat Anda di %s sudah dibuat.

Kode booking : %s
Paket        : %s
Berangkat    : %s
Total        : %s
Pembayaran   : %s
Batas bayar  : %s

Transfer ke:
%s
Setelah transfer, unggah bukti pembayaran lewat halaman Booking Saya:
%s

%s`, nama, brandName, kodeBooking, paket, tglBerangkat, total, dp, batasBayar, b.String(), link, brandName)
	return
}
