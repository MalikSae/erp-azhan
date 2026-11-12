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

// EmailChanged dikirim ke alamat email LAMA setelah email akun diganti, agar pemilik tahu bila bukan dia yang menggantinya.
// Tautan lupa password tidak disertakan: tautan itu kini terkirim ke alamat baru, jadi pemilik diarahkan ke admin.
func EmailChanged(brandName, nama, emailBaru string) (subject, text string) {
	brandName, nama, emailBaru = inline(brandName, maxInlineTeks), inline(nama, maxInlineNama), inline(emailBaru, maxInlineTeks)
	subject = "Email akun " + brandName + " diubah"
	text = fmt.Sprintf(`Halo %s,

Email akun Anda di %s baru saja diubah menjadi %s, dan perangkat lain yang masuk ke akun ini sudah dikeluarkan. Email berikutnya dari kami akan dikirim ke alamat baru tersebut.

Jika ini bukan Anda, segera hubungi admin %s lewat WhatsApp resmi agar akun Anda dipulihkan.

%s`, nama, brandName, emailBaru, brandName, brandName)
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

// PerubahanDataJamaah memberi tahu pemilik akun bahwa pengajuan perubahan data jamaahnya disetujui atau ditolak.
// catatan: alasan penolakan dari admin (dipakai bila ditolak).
func PerubahanDataJamaah(brandName, nama, namaJamaah string, disetujui bool, catatan, link string) (subject, text string) {
	brandName, nama, namaJamaah, link = inline(brandName, maxInlineTeks), inline(nama, maxInlineNama), inline(namaJamaah, maxInlineNama), inline(link, maxInlineTautan)
	if disetujui {
		subject = "Perubahan data jamaah " + namaJamaah + " disetujui"
		text = fmt.Sprintf(`Halo %s,

Perubahan data jamaah %s yang Anda ajukan sudah disetujui dan kini tersimpan.

Lihat data jamaah:

%s

%s`, nama, namaJamaah, link, brandName)
		return
	}
	subject = "Perubahan data jamaah " + namaJamaah + " belum dapat disetujui"
	text = fmt.Sprintf(`Halo %s,

Perubahan data jamaah %s yang Anda ajukan belum dapat disetujui.

Alasan: %s

Data lama tetap berlaku. Anda dapat mengajukan perubahan lagi dari halaman berikut:

%s

%s`, nama, namaJamaah, inline(catatan, maxInlineTeks), link, brandName)
	return
}
