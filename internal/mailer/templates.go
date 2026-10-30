package mailer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Batas panjang nilai yang disisipkan ke email (pertahanan berlapis; validasi utama ada di
// customer.ValidName). Nilai dari pengguna tidak boleh membawa baris baru sehingga tidak bisa
// menyusun paragraf/tautan palsu di tengah email.
const (
	maxInlineNama   = 100
	maxInlineTeks   = 200
	maxInlineTautan = 2048
)

// inline menormalkan nilai satu baris: karakter kontrol (termasuk CR/LF/tab dan pemisah baris
// Unicode) menjadi spasi, spasi beruntun dirapatkan, lalu dipotong ke max rune.
func inline(s string, max int) string {
	var b strings.Builder
	spasi := false
	for _, r := range s {
		if unicode.IsControl(r) || r == ' ' || r == ' ' || unicode.IsSpace(r) {
			if !spasi && b.Len() > 0 {
				b.WriteByte(' ')
			}
			spasi = true
			continue
		}
		spasi = false
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(out) > max {
		out = strings.TrimSpace(string([]rune(out)[:max]))
	}
	return out
}

func VerifyEmail(brandName, nama, link string) (subject, text string) {
	brandName, nama, link = inline(brandName, maxInlineTeks), inline(nama, maxInlineNama), inline(link, maxInlineTautan)
	subject = "Verifikasi email akun " + brandName
	text = fmt.Sprintf(`Halo %s,

Terima kasih sudah mendaftar akun di %s. Buka tautan berikut untuk memverifikasi email Anda (berlaku 24 jam):

%s

Jika Anda tidak merasa mendaftar, abaikan email ini.

%s`, nama, brandName, link, brandName)
	return
}

// PasswordChanged memberi tahu pemilik akun bahwa password diubah dari dalam akun. link = halaman lupa password.
func PasswordChanged(brandName, nama, link string) (subject, text string) {
	brandName, nama, link = inline(brandName, maxInlineTeks), inline(nama, maxInlineNama), inline(link, maxInlineTautan)
	subject = "Password akun " + brandName + " diubah"
	text = fmt.Sprintf(`Halo %s,

Password akun Anda di %s baru saja diubah, dan perangkat lain yang masuk ke akun ini sudah dikeluarkan.

Jika ini bukan Anda, atur ulang password sekarang lewat tautan berikut:

%s

%s`, nama, brandName, link, brandName)
	return
}

func ResetPassword(brandName, nama, link string) (subject, text string) {
	brandName, nama, link = inline(brandName, maxInlineTeks), inline(nama, maxInlineNama), inline(link, maxInlineTautan)
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
	brandName, nama = inline(brandName, maxInlineTeks), inline(nama, maxInlineNama)
	kodeBooking, paket = inline(kodeBooking, maxInlineTeks), inline(paket, maxInlineTeks)
	tglBerangkat, total, dp, batasBayar = inline(tglBerangkat, maxInlineTeks), inline(total, maxInlineTeks), inline(dp, maxInlineTeks), inline(batasBayar, maxInlineTeks)
	link = inline(link, maxInlineTautan)
	subject = fmt.Sprintf("Booking %s berhasil dibuat (%s)", paket, kodeBooking)
	var b strings.Builder
	for _, k := range banks {
		fmt.Fprintf(&b, "  - %s %s a.n. %s\n", inline(k.Bank, maxInlineTeks), inline(k.Nomor, maxInlineTeks), inline(k.AtasNama, maxInlineTeks))
	}
	if b.Len() == 0 {
		b.WriteString("  (hubungi kami untuk informasi rekening tujuan)\n")
	}
	text = fmt.Sprintf(`Halo %s,

Booking seat Anda di %s sudah dibuat.

Kode booking : %s
Paket        : %s
Berangkat    : %s
Total        : %s
Pembayaran   : %s
Batas bayar  : %s

Seat ditahan sampai batas bayar. Lewat dari waktu itu, seat dilepas.

Transfer ke:
%s
Setelah transfer, unggah bukti pembayaran lewat halaman detail booking:
%s

%s`, nama, brandName, kodeBooking, paket, tglBerangkat, total, dp, batasBayar, b.String(), link, brandName)
	return
}
