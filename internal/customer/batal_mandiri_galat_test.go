package customer

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-sql-driver/mysql"
)

// Audit booking 2026-11-26: bentrok kunci saat batal mandiri dijawab 409 (aman dicoba lagi), bukan 500.
func TestGalatBatalMandiri(t *testing.T) {
	cases := []struct {
		nama   string
		err    error
		status int
	}{
		{"tidak memenuhi syarat", errTidakBisaBatal, http.StatusConflict},
		{"deadlock", fmt.Errorf("booking.UpdateStatus: %w", &mysql.MySQLError{Number: 1213}), http.StatusConflict},
		{"lock wait timeout", &mysql.MySQLError{Number: 1205}, http.StatusConflict},
		{"galat lain", errors.New("koneksi putus"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		if status, pesan := galatBatalMandiri(c.err); status != c.status {
			t.Errorf("%s: %d %q, want %d", c.nama, status, pesan, c.status)
		}
	}
}
