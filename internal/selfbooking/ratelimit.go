package selfbooking

import (
	"time"

	"erp-azhan/api/internal/shared"
)

// windowLimiter memakai pembatas bersama (shared.WindowLimiter). Salinan lokal lama memisahkan allow dan
// record, sehingga request paralel lolos bersamaan sebelum satu pun tercatat (audit keamanan 2026-11-12).
// Periksa sekaligus catat lewat Take; kembalikan slot lewat Refund bila kejadian ternyata tidak dihitung.
type windowLimiter = shared.WindowLimiter

func newWindowLimiter(limit int, window time.Duration) *windowLimiter {
	return shared.NewWindowLimiter(limit, window)
}
