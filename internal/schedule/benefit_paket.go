package schedule

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"erp-azhan/api/internal/benefit"
	"erp-azhan/api/internal/shared"
)

// errPublishLampau: paket tidak boleh terbit bila keberangkatan sudah lewat (MP-04).
// errBenefitKurang: paket yang tampil ke publik selalu membawa empat highlight benefit (keputusan 13 November
// 2026). Draf boleh belum lengkap.
const errBenefitKurang = "paket yang dijual wajib punya 4 highlight benefit"

// benefitCukup menolak status published bila jumlah benefit bukan empat. Draf dan arsip tidak diperiksa.
func benefitCukup(w http.ResponseWriter, status string, jumlah int) bool {
	if status == StatusPublished && jumlah != benefit.JumlahPerPaket {
		shared.WriteError(w, http.StatusBadRequest, errBenefitKurang)
		return false
	}
	return true
}

// benefitSah memeriksa daftar benefit yang dipilih; pesan untuk pengguna ditulis sebagai 400.
func (h *Handler) benefitSah(ctx context.Context, w http.ResponseWriter, brandID, scheduleID int64, ids []int64) bool {
	err := h.repo.ValidasiBenefit(ctx, brandID, scheduleID, ids)
	if err == nil {
		return true
	}
	var pe benefit.PilihanError
	if errors.As(err, &pe) {
		shared.WriteError(w, http.StatusBadRequest, pe.Error())
		return false
	}
	log.Printf("[ERROR] schedule: validasi benefit: %v", err)
	shared.WriteError(w, http.StatusInternalServerError, "Gagal memeriksa highlight benefit.")
	return false
}

// ValidasiBenefit memeriksa daftar highlight benefit sebuah paket (milik brand paket, tidak kembar, paling
// banyak empat, benefit nonaktif hanya bila paket sudah memakainya). scheduleID 0 untuk paket baru.
func (r *Repository) ValidasiBenefit(ctx context.Context, brandID, scheduleID int64, ids []int64) error {
	return benefit.ValidasiPilihan(ctx, r.db, brandID, scheduleID, ids)
}

// JumlahBenefit menghitung highlight benefit yang tersimpan pada sebuah paket.
func (r *Repository) JumlahBenefit(ctx context.Context, scheduleID int64) (int, error) {
	n, err := benefit.JumlahTautan(ctx, r.db, scheduleID)
	if err != nil {
		return 0, fmt.Errorf("schedule.JumlahBenefit: %w", err)
	}
	return n, nil
}
