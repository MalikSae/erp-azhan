package selfbooking

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestBentrokTransaksiJadi409(t *testing.T) {
	for _, no := range []uint16{1213, 1205} {
		rec := httptest.NewRecorder()
		tulisGalatBooking(rec, fmt.Errorf("proses: %w", &mysql.MySQLError{Number: no}), false)
		if rec.Code != http.StatusConflict {
			t.Fatalf("mysql %d: status %d, body %s", no, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	tulisGalatBooking(rec, &mysql.MySQLError{Number: 1062}, false)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("1062 harus tetap 500, dapat %d", rec.Code)
	}
}
