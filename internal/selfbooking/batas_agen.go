package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-azhan/api/internal/agen"
)

// KodeBatasSeatAgen: kode stabil jawaban 429 saat batas hold seat agen tercapai.
const KodeBatasSeatAgen = "BATAS_SEAT_AGEN"

// ErrBatasSeatAgen dipakai lewat errors.Is; detail sisa seat ada di *BatasSeatAgenError.
var ErrBatasSeatAgen = errors.New("batas hold seat agen tercapai")

// BatasSeatAgenError membawa sisa seat agar agen tahu berapa yang masih bisa ditahan.
type BatasSeatAgenError struct {
	BatasPerJadwal, BatasTotal int
	SisaPerJadwal, SisaTotal   int
}

func (e *BatasSeatAgenError) Error() string {
	sisa := e.SisaPerJadwal
	if e.SisaTotal < sisa {
		sisa = e.SisaTotal
	}
	return fmt.Sprintf("Batas hold seat agen tercapai: Anda masih bisa menahan %d seat di jadwal ini (batas %d per jadwal, %d total). Minta DP dari jamaah yang sudah terdaftar agar seat terlepas dari hitungan, atau hubungi admin untuk menaikkan batas.",
		sisa, e.BatasPerJadwal, e.BatasTotal)
}

func (e *BatasSeatAgenError) Is(target error) bool { return target == ErrBatasSeatAgen }

// seatDibutuhkan: seat yang dikurangi booking ini (PIC + anggota dewasa; bayi tidak memakai seat).
func seatDibutuhkan(req BookingRequest) int {
	n := 1
	for _, a := range req.Anggota {
		if a.PaxType == "reguler" {
			n++
		}
	}
	return n
}

// cekBatasSeatAgen mengunci baris akun agen, lalu memastikan seat yang ditahan booking buatannya (yang belum DP)
// ditambah booking ini tidak melewati batas per jadwal dan total (keputusan 12 November 2026).
func cekBatasSeatAgen(ctx context.Context, tx *sql.Tx, agenID, scheduleID int64, butuh int) error {
	perJadwal, total, err := agen.KunciBatasSeatAgen(ctx, tx, agenID)
	if err != nil {
		return err
	}
	ditahan, diJadwal, err := agen.SeatDitahanAgen(ctx, tx, agenID, scheduleID)
	if err != nil {
		return err
	}
	if diJadwal+butuh > perJadwal || ditahan+butuh > total {
		return &BatasSeatAgenError{
			BatasPerJadwal: perJadwal, BatasTotal: total,
			SisaPerJadwal: max(perJadwal-diJadwal, 0), SisaTotal: max(total-ditahan, 0),
		}
	}
	return nil
}
