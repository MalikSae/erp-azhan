package selfbooking

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/mailer"
)

// PathDetailBooking adalah rute halaman detail booking di situs brand (azhan-booking). Rute tetap
// memakai kata lama "pesanan"; hanya teks tampilan yang memakai istilah "booking".
const PathDetailBooking = "/booking/pesanan/"

// mailInfo adalah data brand dan jadwal yang dibutuhkan email "Booking berhasil dibuat".
type mailInfo struct {
	BrandName  string
	Domain     string
	Paket      string
	Berangkat  time.Time
}

// loadMailInfo membaca nama brand, domain situs, nama paket, dan tanggal berangkat dari database.
func (r *Repository) loadMailInfo(ctx context.Context, brandID, scheduleID int64) (*mailInfo, error) {
	name, domain, err := customer.NewRepository(r.db).BrandSite(ctx, brandID)
	if err != nil {
		return nil, err
	}
	info := &mailInfo{BrandName: name, Domain: domain}
	err = r.db.QueryRowContext(ctx, `SELECT jadwal_nama, berangkat_tanggal FROM schedules WHERE id=? AND brand_id=?`, scheduleID, brandID).Scan(&info.Paket, &info.Berangkat)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return info, nil
}

var bulanPanjang = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

var wib = time.FixedZone("WIB", 7*3600)

func formatTanggal(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%d %s %d", t.Day(), bulanPanjang[t.Month()-1], t.Year())
}

func formatRupiah(v float64) string {
	s := strconv.FormatInt(int64(v+0.5), 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "Rp " + b.String()
}

// buildBookingEmail menyusun email "Booking berhasil dibuat" dari data booking nyata. Ia mengembalikan
// false bila data minimum (alamat tujuan, domain situs) tidak ada.
func buildBookingEmail(to, nama string, info *mailInfo, resp *BookingResponse) (mailer.Message, bool) {
	to = strings.TrimSpace(to)
	if to == "" || info == nil || info.Domain == "" || resp == nil {
		return mailer.Message{}, false
	}
	b := resp.Booking
	pembayaran := "DP minimal " + formatRupiah(b.MinimalDP)
	if b.MinimalDP >= b.TotalHarga {
		pembayaran = "Bayar penuh " + formatRupiah(b.TotalHarga)
	}
	batas := "-"
	if t, err := time.Parse(time.RFC3339, b.SeatHoldExpiresAt); err == nil {
		t = t.In(wib)
		batas = fmt.Sprintf("%s %s WIB", formatTanggal(t), t.Format("15:04"))
	}
	var banks []mailer.BankInfo
	for _, k := range resp.BankAccounts {
		banks = append(banks, mailer.BankInfo{Bank: k.BankName, Nomor: k.AccountNumber, AtasNama: k.AccountHolder})
	}
	link := customer.SiteLink(info.Domain, PathDetailBooking+strconv.FormatInt(b.BookingID, 10), nil)
	subject, text := mailer.OrderCreated(info.BrandName, nama, b.BookingCode, info.Paket, formatTanggal(info.Berangkat), formatRupiah(b.TotalHarga), pembayaran, batas, link, banks)
	return mailer.Message{To: to, Subject: subject, Text: text}, true
}

// notifyBookingCreated mengirim email konfirmasi secara asinkron setelah booking mandiri BARU dibuat.
// Kegagalan hanya dicatat di log dan tidak mempengaruhi booking maupun respons ke pelanggan.
func (h *Handler) notifyBookingCreated(account *customer.Account, scheduleID int64, resp *BookingResponse) {
	if h.mail == nil || resp == nil || resp.Replayed {
		return
	}
	h.mailWG.Add(1)
	go func() {
		defer h.mailWG.Done()
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[ERROR] selfbooking mail: panic %v", rec)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		load := h.loadInfo
		if load == nil {
			load = h.repo.loadMailInfo
		}
		info, err := load(ctx, account.BrandID, scheduleID)
		if err != nil {
			log.Printf("[WARN] selfbooking: data email booking %s gagal dimuat: %v", resp.Booking.BookingCode, err)
			return
		}
		msg, ok := buildBookingEmail(account.Email, account.NamaLengkap, info, resp)
		if !ok {
			log.Printf("[WARN] selfbooking: email booking %s dilewati (alamat atau domain situs kosong)", resp.Booking.BookingCode)
			return
		}
		if err := h.mail.Send(ctx, msg); err != nil {
			log.Printf("[ERROR] mailer: %v", err)
		}
	}()
}
